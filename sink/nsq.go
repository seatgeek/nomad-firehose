package sink

import (
	"fmt"
	"os"
	"time"

	"github.com/nsqio/go-nsq"
	log "github.com/sirupsen/logrus"
)

type NSQSink struct {
	producer  *nsq.Producer
	topicName string
	stopCh    chan interface{}
	putCh     chan []byte
}

func NewNSQ() (*NSQSink, error) {

	addrNSQ := os.Getenv("SINK_NSQ_ADDR")
	if addrNSQ == "" {
		return nil, fmt.Errorf("[sink/nsq] Missing SINK_NSQ_ADDR (example: 127.0.0.1:4150)")
	}
	log.Infof("[sink/nsq] SINK_NSQ_ADDR=%s", addrNSQ)

	topicName := os.Getenv("SINK_NSQ_TOPIC_NAME")
	if topicName == "" {
		return nil, fmt.Errorf("[sink/nsq] Missing SINK_NSQ_TOPIC_NAME (example: nomad-firehose)")
	}
	log.Infof("[sink/nsq] SINK_NSQ_TOPIC_NAME=%s", topicName)

	conf := nsq.NewConfig()
	producer, err := nsq.NewProducer(addrNSQ, conf)
	if err != nil {
		return nil, fmt.Errorf("[sink/nsq] Failed to connect to NSQ: %v", err)
	}

	return &NSQSink{
		producer:  producer,
		topicName: topicName,
		stopCh:    make(chan interface{}),
		putCh:     make(chan []byte, 1000),
	}, nil
}

func (s *NSQSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

	// have 1 writer to NSQ
	go s.write(1)

	// wait forever for a stop signal to happen
	for {
		select {
		case <-s.stopCh:
			break
		}
		break
	}

	return nil
}

func (s *NSQSink) Stop() {
	log.Infof("[sink/nsq] ensure write queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Infof("[sink/nsq] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
}

func (s *NSQSink) Put(data []byte) error {
	s.putCh <- data

	return nil
}

func (s *NSQSink) write(id int) {
	log.Infof("[sink/nsq/%d] Starting writer", id)

	for {
		select {
		case data := <-s.putCh:
			if err := s.producer.Publish(s.topicName, data); err != nil {
				log.Infof("[sink/nsq/%d] %s", id, err)
			} else {
				log.Infof("[sink/nsq/%d] Publish OK", id)
			}
		}
	}
}
