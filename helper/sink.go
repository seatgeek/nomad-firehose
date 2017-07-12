package helper

import (
	"fmt"
	"os"

	"github.com/seatgeek/nomad-firehose/sink"
	"github.com/seatgeek/nomad-firehose/structs"
)

// GetSink ...
func GetSink() (structs.Sink, error) {
	sinkType := os.Getenv("SINK_TYPE")
	if sinkType == "" {
		return nil, fmt.Errorf("Missing SINK_TYPE: amqp, kinesis or stdout")
	}

	switch sinkType {
	case "amqp":
		fallthrough
	case "rabbitmq":
		return sink.NewRabbitmq()
	case "kinesis":
		return sink.NewKinesis()
	case "stdout":
		return sink.NewStdout()
	default:
		return nil, fmt.Errorf("Invalid SINK_TYPE: amqp, kinesis or stdout")
	}
}
