package sink

import (
	"fmt"
	"os"
)

// GetSink ...
func GetSink() (Sink, error) {
	sinkType := os.Getenv("SINK_TYPE")
	if sinkType == "" {
		return nil, fmt.Errorf("Missing SINK_TYPE: amqp, kafka, kinesis, nsq, rabbitmq, redis, mongodb, http or stdout")
	}

	switch sinkType {
	case "amqp":
		return NewRabbitmq()
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
	case "mongodb":
		return NewMongodb()
	case "http":
		return NewHttp()
	case "stdout":
		return NewStdout()
	default:
		return nil, fmt.Errorf("Invalid SINK_TYPE: %s, Valid values: amqp, kafka, kinesis, nsq, rabbitmq, redis, mongodb, http or stdout", sinkType)
	}
}
