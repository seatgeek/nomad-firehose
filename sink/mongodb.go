package sink

import (
	"strconv"
	"time"

	"os"

	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// MongodbSink ...
type MongodbSink struct {
	session     *mgo.Session
	database    string
	collection  string
	idField     string
	workerCount int
	timestamps  bool
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

	idField := os.Getenv("SINK_MONGODB_ID")

	workerCountStr := os.Getenv("SINK_MONGODB_WORKERS")
	if workerCountStr == "" {
		workerCountStr = "1"
	}

	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid SINK_MONGODB_WORKERS, must be an integer")
	}

	if err != nil {
		return nil, fmt.Errorf("Invalid SINK_MONGODB_WORKERS, must be an integer")
	}

	session, err := mgo.Dial(connStr)
	if err != nil {
		return nil, fmt.Errorf("[sink/mongodb] failed to connect to string: %s", err)
	}


	return &MongodbSink{
		session:     session,
		database:    database,
		collection:  collection,
		idField:     idField,
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
	defer s.session.Close()
}

// Put ..
func (s *MongodbSink) Put(data []byte) error {
	s.putCh <- data

	return nil
}

func (s *MongodbSink) write(id int) {
	log.Infof("[sink/mongodb/%d] Starting writer", id)

	c := s.session.DB(s.database).C(s.collection)

	for {
		select {
		case data := <-s.putCh:
			var record bson.M
			err := bson.UnmarshalJSON(data, &record)
			if (err != nil) {
				log.Errorf("[sink/mongodb/%d] %s", id, err)
				continue
			}

			update := bson.M{"$set": record}
			if (s.idField != "") {
				id := record[s.idField].(string)
				if (id == "") {
					log.Errorf("[sink/mongodb/%d] missing id field $s", id, s.idField)
					continue
				}
				_, err = c.UpsertId(bson.ObjectIdHex(id), update)
			} else {
				err = c.Insert(update)
			}

			if err != nil {
				log.Errorf("[sink/mongodb/%d] %s", id, err)
			} else {
				log.Debugf("[sink/mongodb/%d] publish ok", id)
			}
		}
	}
}
