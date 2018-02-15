package sink

import (
	"fmt"
	"os"
)

// GetSink ...
func GetSink() (Sink, error) {
	sinkType := os.Getenv("SINK_TYPE")
	if sinkType == "" {
		return nil, fmt.Errorf("Missing SINK_TYPE: amqp, kafka, kinesis, nsq, redis or stdout")
	}

	switch sinkType {
	case "amqp":
		fallthrough
	case "rabbitmq":
		return NewRabbitmq()
	case "kafka":
		return NewKafka()
	case "kinesis":
		return NewKinesis()
	case "stdout":
		return NewStdout()
	case "nsq":
		return NewNSQ()
	case "redis":
		return NewRedis()
	default:
		return nil, fmt.Errorf("Invalid SINK_TYPE: %s, Valid values: amqp, kafka, kinesis, nsq, redis or stdout", sinkType)
	}
}
