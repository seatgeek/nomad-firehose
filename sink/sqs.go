package sink

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	log "github.com/sirupsen/logrus"
)

// SQS ...
type SQSSink struct {
	session   *session.Session
	sqs       *sqs.SQS
	queueName string
	groupId   string
	stopCh    chan interface{}
	putCh     chan []byte
	batchCh   chan [][]byte
}

// NNewSQS ...
func NewSQS(groupId string) (*SQSSink, error) {
	queueName := os.Getenv("SINK_SQS_QUEUE_URL")
	if queueName == "" {
		return nil, fmt.Errorf("[sink/sqs] Missing SINK_SQS_QUEUE_URL")
	}

	sess := session.Must(session.NewSession())
	svc := sqs.New(sess)

	return &SQSSink{
		session:   sess,
		sqs:       svc,
		queueName: queueName,
		groupId:   groupId,
		stopCh:    make(chan interface{}),
		putCh:     make(chan []byte, 10000),
		batchCh:   make(chan [][]byte, 100),
	}, nil
}

// Start ...
func (s *SQSSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

	go s.batch()
	go s.write()

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
func (s *SQSSink) Stop() {
	log.Infof("[sink/sqs] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Infof("[sink/sqs] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
}

// Put ..
func (s *SQSSink) Put(data []byte) error {
	s.putCh <- data

	log.Infof("[sink/sqs] (%d messages left)", len(s.putCh))

	return nil
}

func (s *SQSSink) batch() {
	buffer := make([][]byte, 0)
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case data := <-s.putCh:
			if len(buffer) == 10 {
				s.batchCh <- buffer
				buffer = make([][]byte, 0)
			}

			buffer = append(buffer, data)

		case _ = <-ticker.C:

			// If we had any accumulated messages from the ticker, drop them
			for {
				if len(ticker.C) > 0 {
					_ = <-ticker.C
				} else {
					break
				}
			}

			// If there is anything else in the putCh, wait a little longer
			if len(s.putCh) > 0 {
				continue
			}

			if len(buffer) > 0 {
				s.batchCh <- buffer
				buffer = make([][]byte, 0)
			}

		}
	}
}

func (s *SQSSink) write() {
	log.Infof("[sink/sqs] Starting writer")

	var id int64

	for {
		select {
		case batch := <-s.batchCh:
			entries := make([]*sqs.SendMessageBatchRequestEntry, 0)

			for _, data := range batch {
				mID := aws.String(strconv.FormatInt(id, 10))
				entry := &sqs.SendMessageBatchRequestEntry{
					Id:                     mID,
					MessageBody:            aws.String(string(data)),
					MessageGroupId:         aws.String(s.groupId),
					MessageDeduplicationId: mID,
				}

				entries = append(entries, entry)
				id = id + 1
			}

			_, err := s.sqs.SendMessageBatch(&sqs.SendMessageBatchInput{
				Entries:  entries,
				QueueUrl: aws.String(s.queueName),
			})

			if err != nil {
				log.Errorf("[sink/sqs] %s", err)
			} else {
				log.Infof("[sink/sqs] queued %d messages", len(batch))
			}
		}
	}
}
