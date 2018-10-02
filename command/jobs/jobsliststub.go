package jobs

import (
	"encoding/json"

	nomad "github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
)

// Firehose ...
type JobListStubFirehose struct {
	FirehoseBase
}

// NewFirehose ...
func NewJobListStubFirehose() (*JobListStubFirehose, error) {
	base, err := NewFirehoseBase()
	if err != nil {
		return nil, err
	}

	return &JobListStubFirehose{FirehoseBase: *base}, nil
}

func (f *JobListStubFirehose) Name() string {
	return "jobliststub"
}

func (f *JobListStubFirehose) Start() {
	f.FirehoseBase.Start(f.watchJobList)
}

// Publish an update from the firehose
func (f *JobListStubFirehose) Publish(update *nomad.JobListStub) {
	b, err := json.Marshal(update)
	if err != nil {
		log.Error(err)
	}

	f.sink.Put(b)
}

func (f *JobListStubFirehose) watchJobList(job *nomad.JobListStub) {
	f.Publish(job)
}


