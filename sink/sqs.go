package sink

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	queueName := os.Getenv("SINK_SQS_QUEUE_NAME")

	if queueName == "" {
		return nil, fmt.Errorf("[sink/sqs] Missing SINK_SQS_QUEUE_NAME")
	}

	sess := session.Must(session.NewSession())
	svc := sqs.New(sess)

	output, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})

	if err != nil {
		return nil, fmt.Errorf("Failed to find queue: %s", err)
	}

	return &SQSSink{
		session:   sess,
		sqs:       svc,
		queueName: *output.QueueUrl,
		groupId:   groupId,
		stopCh:    make(chan interface{}),
		putCh:     make(chan []byte, 1000),
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

	return nil
}

func (s *SQSSink) batch() {
	buffer := make([][]byte, 0)
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case data := <-s.putCh:
			buffer = append(buffer, data)

			if len(buffer) == 10 {
				s.batchCh <- buffer
				buffer = make([][]byte, 0)
			}

		case _ = <-ticker.C:
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

			err := s.sendBatch(entries)
			if err != nil && strings.Contains(err.Error(), "AWS.SimpleQueueService.BatchRequestTooLong") {
				for i := 0; i < len(entries); i += i {
					err = s.sendBatch([]*sqs.SendMessageBatchRequestEntry{entries[i]})
					if err != nil {
						log.Errorf("[sink/sqs] Retry failed for %d: %s", i, err)
					} else {
						log.Infof("[sink/sqs] Retry succeeded for %d", i)
					}
				}

				continue
			}

			if err != nil {
				log.Errorf("[sink/sqs] %s", err)
			} else {
				log.Infof("[sink/sqs] queued %d messages", len(batch))
			}
		}
	}
}

func (s *SQSSink) sendBatch(entries []*sqs.SendMessageBatchRequestEntry) error {
	_, err := s.sqs.SendMessageBatch(&sqs.SendMessageBatchInput{
		Entries:  entries,
		QueueUrl: aws.String(s.queueName),
	})

	return err
}
