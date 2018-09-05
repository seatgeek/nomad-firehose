package jobs

import (
	"encoding/json"

	nomad "github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
)

// Firehose ...
type Firehose struct {
	FirehoseBase
}

// NewFirehose ...
func NewFirehose() (*Firehose, error) {
	base, err := NewFirehoseBase()
	if err != nil {
		return nil, err
	}

	return &Firehose{FirehoseBase: *base}, nil
}


// Publish an update from the firehose
func (f *FirehoseBase) Publish(update *nomad.Job) {
	b, err := json.Marshal(update)
	if err != nil {
		log.Error(err)
	}

	f.sink.Put(b)
}

func (f *Firehose) watch() {
	go f.FirehoseBase.watch()

	for {
		select {
		case job := <- f.jobListSink:
			go func(jobID string) {
				fullJob, _, err := f.nomadClient.Jobs().Info(jobID, &nomad.QueryOptions{})
				if err != nil {
					log.Errorf("Could not read job %s: %s", jobID, err)
					return
				}

				f.Publish(fullJob)
			}(job.ID)
		}
	}
}
