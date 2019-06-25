package sink

import (
	"fmt"
	"os"
)

// GetSink ...
func GetSink(resourceName string) (Sink, error) {
	sinkType := os.Getenv("SINK_TYPE")
	if sinkType == "" {
		return nil, fmt.Errorf("Missing SINK_TYPE: amqp, http, kafka, kinesis, mongodb, nsq, rabbitmq, redis, sqs, stdout, syslog")
	}

	switch sinkType {
	case "amqp":
		return NewRabbitmq()
	case "http":
		return NewHttp()
	case "kafka":
		return NewKafka()
	case "kinesis":
		return NewKinesis()
	case "mongodb":
		return NewMongodb()
	case "nsq":
		return NewNSQ()
	case "rabbitmq":
		return NewRabbitmq()
	case "redis":
		return NewRedis()
	case "stdout":
		return NewStdout()
	case "syslog":
		return NewSyslog()
	case "sqs":
		return NewSQS(resourceName)
	default:
		return nil, fmt.Errorf("Invalid SINK_TYPE: %s, Valid values: amqp, http, kafka, kinesis, mongodb, nsq, rabbitmq, redis, sqs, stdout, syslog", sinkType)
	}
}
