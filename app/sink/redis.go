package sink

import (
	"fmt"
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
	log "github.com/sirupsen/logrus"
)

// RedisSink ...
type RedisSink struct {
	pool   *redis.Pool
	key    string
	stopCh chan interface{}
	putCh  chan []byte
}

// NewStdout ...
func NewRedis() (*RedisSink, error) {
	redisURL := os.Getenv("SINK_REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("[sink/redis] Missing SINK_REDIS_URL (example: redis://[user]:[password]@127.0.0.1[:5672]/0)")
	}

	redisKey := os.Getenv("SINK_REDIS_KEY")
	if redisKey == "" {
		return nil, fmt.Errorf("[sink/redis] Missing SINK_REDIS_KEY (example: my-key")
	}

	redisPool := redis.Pool{
		MaxIdle:     2,
		MaxActive:   2,
		IdleTimeout: time.Minute,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(redisURL) },
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return &RedisSink{
		pool:   &redisPool,
		key:    redisKey,
		stopCh: make(chan interface{}),
		putCh:  make(chan []byte, 1000),
	}, nil
}

// Start ...
func (s *RedisSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

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
func (s *RedisSink) Stop() {
	log.Debugf("[sink/redis] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Debugf("[sink/redis] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
	defer s.pool.Close()
}

// Put ..
func (s *RedisSink) Put(data []byte) error {
	s.putCh <- data
	return nil
}

func (s *RedisSink) write() {
	log.Infof("[sink/redis] Starting writer to key '%s'", s.key)

	for {
		select {
		case data := <-s.putCh:
			conn := s.pool.Get()
			if _, err := conn.Do("RPUSH", s.key, data); err != nil {
				log.Infof("[sink/redis] %s", err)
			} else {
				log.Infof("[sink/redis] Published to key '%s'", s.key)
			}
			conn.Close()
		}
	}
}
