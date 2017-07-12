package sink

import (
	"fmt"
	"os"
)

// GetSink ...
func GetSink() (Sink, error) {
	sinkType := os.Getenv("SINK_TYPE")
	if sinkType == "" {
		return nil, fmt.Errorf("Missing SINK_TYPE: amqp, kinesis or stdout")
	}

	switch sinkType {
	case "amqp":
		fallthrough
	case "rabbitmq":
		return NewRabbitmq()
	case "kinesis":
		return NewKinesis()
	case "stdout":
		return NewStdout()
	default:
		return nil, fmt.Errorf("Invalid SINK_TYPE: amqp, kinesis or stdout")
	}
}
