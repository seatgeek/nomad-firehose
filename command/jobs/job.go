package jobs

import (
	"encoding/json"

	nomad "github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
)

// Firehose ...
type JobFirehose struct {
	FirehoseBase
}

// NewFirehose ...
func NewJobFirehose() (*JobFirehose, error) {
	base, err := NewFirehoseBase()
	if err != nil {
		return nil, err
	}

	return &JobFirehose{FirehoseBase: *base}, nil
}


func (f *JobFirehose) Name() string {
	return "jobs"
}

// Publish an update from the firehose
func (f *JobFirehose) Publish(update *nomad.Job) {
	b, err := json.Marshal(update)
	if err != nil {
		log.Error(err)
	}

	f.sink.Put(*update.ID, b)
}


func (f *JobFirehose) Start() {
	f.FirehoseBase.Start(f.watchJobList)
}

func (f *JobFirehose) watchJobList(job *nomad.JobListStub) {
	go func(jobID string) {
		fullJob, _, err := f.nomadClient.Jobs().Info(jobID, &nomad.QueryOptions{})
		if err != nil {
			log.Errorf("Could not read job %s: %s", jobID, err)
			return
		}

		f.Publish(fullJob)
	}(job.ID)
}
