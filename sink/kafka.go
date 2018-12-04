package sink

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

// KafkaSink ...
type KafkaSink struct {
	// Kafka brokers to send metrics to
	Brokers []string
	// Kafka topic
	Topic string

	producer sarama.SyncProducer

	stopCh chan interface{}
	putCh chan Data
}

func createTlsConfiguration() (t *tls.Config) {
	caFile := os.Getenv("SINK_KAFKA_CA_CERT_PATH")
	if caFile != "" {
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			RootCAs:            caCertPool,
		}
	}

	return t
}

// NewKafka ...
func NewKafka() (*KafkaSink, error) {
	brokers := os.Getenv("SINK_KAFKA_BROKERS")
	if brokers == "" {
		return nil, fmt.Errorf("[sink/kafka] Missing SINK_KAFKA_BROKERS")
	}

	brokerList := strings.Split(brokers, ",")
	log.Debugf("[sink/kafka] Kafka brokers: %s", strings.Join(brokerList, ", "))

	topic := os.Getenv("SINK_KAFKA_TOPIC")
	if topic == "" {
		return nil, fmt.Errorf("[sink/kafka] Missing SINK_KAFKA_TOPIC")
	}
	log.Debugf("[sink/kafka] Kafka topic: %s", topic)

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	tlsConfig := createTlsConfiguration()
	if tlsConfig != nil {
		config.Net.TLS.Config = tlsConfig
		config.Net.TLS.Enable = true
	}

	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	return &KafkaSink{
		Brokers:  brokerList,
		Topic:    topic,
		producer: producer,
		stopCh:   make(chan interface{}),
		putCh:    make(chan Data, 1000),
	}, nil
}

// Start ...
func (s *KafkaSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

	go s.write()

	return nil
}

// Stop ...
func (s *KafkaSink) Stop() {
	log.Debugf("[sink/kafka] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Debugf("[sink/kafka] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
}

// Put ..
func (s *KafkaSink) Put(key string, value []byte) error {
	s.putCh <- Data{key, value}

	return nil
}

func (s *KafkaSink) write() {
	log.Info("[sink/kafka] Starting writer")

	for {
		select {
		case data := <-s.putCh:
			message := &sarama.ProducerMessage{Topic: s.Topic}
			message.Value = sarama.StringEncoder(string(data.value))
			message.Key = sarama.StringEncoder(data.key)
			partition, offset, err := s.producer.SendMessage(message)
			if err != nil {
				log.Errorf("Failed to produce message: %s", err)
			} else {
				log.Debugf("[sink/kafka] topic=%s\tpartition=%d\toffset=%d\n", s.Topic, partition, offset)
			}
		}
	}
}
