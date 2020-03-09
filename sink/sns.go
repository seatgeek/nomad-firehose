package sink

import (
	"strconv"
	"time"

	"os"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	log "github.com/sirupsen/logrus"
)

// SNSSink ...
type SNSSink struct {
	session     *session.Session
	sns         *sns.SNS
	topicArn    string
	stopCh      chan interface{}
	putCh       chan []byte
	workerCount int
}

// NewSNS ...
func NewSNS() (*SNSSink, error) {
	topicArn := os.Getenv("SINK_SNS_TOPIC_ARN")
	if topicArn == "" {
		return nil, fmt.Errorf("[sink/sns] Missing SINK_SNS_TOPIC_ARN")
	}

	workerCountStr := os.Getenv("SINK_SNS_WORKERS")
	if workerCountStr == "" {
		workerCountStr = "1"
	}

	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SINK_SNS_WORKERS, must be an integer")
	}

	sess := session.Must(session.NewSession())
	svc := sns.New(sess)

	return &SNSSink{
		session:     sess,
		sns:         svc,
		topicArn:    topicArn,
		stopCh:      make(chan interface{}),
		putCh:       make(chan []byte, 1000),
		workerCount: workerCount,
	}, nil
}

// Start ...
func (s *SNSSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

	for i := 0; i < s.workerCount; i++ {
		go s.write(i)
	}

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

// Stop ...
func (s *SNSSink) Stop() {
	log.Infof("[sink/sns] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Info("[sink/sns] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
}

// Put ..
func (s *SNSSink) Put(data []byte) error {
	s.putCh <- data

	return nil
}

func (s *SNSSink) write(id int) {
	log.Infof("[sink/sns/%d] Starting writer", id)

	topicArn := aws.String(s.topicArn)

	for {
		select {
		case data := <-s.putCh:
			message := aws.String(string(data))
			putOutput, err := s.sns.Publish(&sns.PublishInput{
				Message:  message,
				TopicArn: topicArn,
			})

			if err != nil {
				log.Errorf("[sink/sns/%d] %s", id, err)
			} else {
				log.Infof("[sink/sns/%d] %v", id, putOutput)
			}
		}
	}
}

// Name ..
func (s *SNSSink) Name() string {
	return "sns"
}
