package sink

import (
	"context"
	"strconv"
	"time"

	"os"

	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	mongo "github.com/mongodb/mongo-go-driver/mongo"
)

// MongodbSink ...
type MongodbSink struct {
	conn        *mongo.Client
	database    string
	collection  string
	workerCount int
	stopCh      chan interface{}
	putCh       chan []byte
}

// NewMongodb ...
func NewMongodb() (*MongodbSink, error) {
	connStr := os.Getenv("SINK_MONGODB_CONNECTION")
	if connStr == "" {
		return nil, fmt.Errorf("[sink/mongodb] Missing SINK_MONGODB_CONNECTION (example: mongodb://foo:bar@localhost:27017)")
	}

	database := os.Getenv("SINK_MONGODB_DATABASE")
	if database == "" {
		return nil, fmt.Errorf("[sink/mongodb] Mising SINK_MONGODB_DATABASE")
	}

	collection := os.Getenv("SINK_MONGODB_COLLECTION")
	if collection == "" {
		return nil, fmt.Errorf("[sink/mongodb] Missing SINK_MONGODB_COLLECTION")
	}

	workerCountStr := os.Getenv("SINK_MONGODB_WORKERS")
	if workerCountStr == "" {
		workerCountStr = "1"
	}
	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid SINK_MONGODB_WORKERS, must be an integer")
	}

	conn, err := mongo.NewClient(connStr)
	if err != nil {
		return nil, fmt.Errorf("[sink/mongodb] Invalid to connect to string: %s", err)
	}

	err = conn.Connect(context.Background())
	if err != nil {
		return nil, fmt.Errorf("[sink/mongodb] failed to connect to string: %s", err)
	}

	return &MongodbSink{
		conn:        conn,
		database:    database,
		collection:  collection,
		workerCount: workerCount,
		stopCh:      make(chan interface{}),
		putCh:       make(chan []byte, 1000),
	}, nil
}

// Start ...
func (s *MongodbSink) Start() error {
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
func (s *MongodbSink) Stop() {
	log.Infof("[sink/mongodb] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Info("[sink/mongodb] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
	defer s.conn.Disconnect(context.Background())
}

// Put ..
func (s *MongodbSink) Put(data []byte) error {
	s.putCh <- data

	return nil
}

func (s *MongodbSink) write(id int) {
	log.Infof("[sink/mongodb/%d] Starting writer", id)

	collection := s.conn.Database(s.database).Collection(s.collection)

	for {
		select {
		case data := <-s.putCh:
			m := make(map[string]interface{})
			err := json.Unmarshal(data, &m)

			if err != nil {
				log.Errorf("[sink/mongodb/%d] %s", id, err)
				continue
			}
			_, err = collection.InsertOne(context.Background(), m)
			if err != nil {
				log.Errorf("[sink/mongodb/%d] %s", id, err)
			} else {
				log.Debugf("[sink/mongodb/%d] publish ok", id)
			}
		}
	}
}
