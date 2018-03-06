package sink

import (
	"fmt"
	"os"
)

// GetSink ...
func GetSink() (Sink, error) {
	sinkType := os.Getenv("SINK_TYPE")
	if sinkType == "" {
		return nil, fmt.Errorf("Missing SINK_TYPE: amqp, kafka, kinesis, nsq, rabbitmq, redis or stdout")
	}

	switch sinkType {
	case "amqp":
		fallthrough
	case "kafka":
		return NewKafka()
	case "kinesis":
		return NewKinesis()
	case "nsq":
		return NewNSQ()
	case "rabbitmq":
		return NewRabbitmq()
	case "redis":
		return NewRedis()
	case "stdout":
		return NewStdout()
	default:
		return nil, fmt.Errorf("Invalid SINK_TYPE: %s, Valid values: amqp, kafka, kinesis, nsq, rabbitmq, redis or stdout", sinkType)
	}
}
