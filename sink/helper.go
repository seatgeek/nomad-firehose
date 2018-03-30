package sink

import (
	"fmt"
	"os"
)

// GetSink ...
func GetSink() (Sink, error) {
	sinkType := os.Getenv("SINK_TYPE")
	if sinkType == "" {
		return nil, fmt.Errorf("Missing SINK_TYPE: amqp, kafka, kinesis, nsq, rabbitmq, redis, d3slack or stdout")
	}

	switch sinkType {
	case "amqp":
		return NewRabbitmq()
	case "d3slack":
		return NewD3Slack()
	case "rabbitmq":
		return NewRabbitmq()
	case "kafka":
		return NewKafka()
	case "kinesis":
		return NewKinesis()
	case "nsq":
		return NewNSQ()
	case "redis":
		return NewRedis()
	case "stdout":
		return NewStdout()
	default:
		return nil, fmt.Errorf("Invalid SINK_TYPE: %s, Valid values: amqp, kafka, kinesis, nsq, rabbitmq, redis or stdout", sinkType)
	}
}
