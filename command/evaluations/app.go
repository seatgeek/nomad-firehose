package evaluations

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-firehose/sink"
	log "github.com/sirupsen/logrus"
)

// Firehose ...
type Firehose struct {
	lastChangeIndex  uint64
	lastChangeTimeCh chan interface{}
	nomadClient      *nomad.Client
	sink             sink.Sink
	stopCh           chan struct{}
}

// NewFirehose ...
func NewFirehose() (*Firehose, error) {
	nomadClient, err := nomad.NewClient(nomad.DefaultConfig())
	if err != nil {
		return nil, err
	}

	sink, err := sink.GetSink()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	return &Firehose{
		nomadClient:      nomadClient,
		sink:             sink,
		stopCh:           make(chan struct{}, 1),
		lastChangeTimeCh: make(chan interface{}, 1),
	}, nil
}

func (f *Firehose) Name() string {
	return "evaluations"
}

func (f *Firehose) UpdateCh() <-chan interface{} {
	return f.lastChangeTimeCh
}

func (f *Firehose) SetRestoreValue(restoreValue interface{}) error {
	switch restoreValue.(type) {
	case int:
		f.lastChangeIndex = uint64(restoreValue.(int))
	case int64:
		f.lastChangeIndex = uint64(restoreValue.(int64))
	default:
		return fmt.Errorf("Unknown restore type '%T' with value '%+v'", restoreValue, restoreValue)
	}
	return nil
}

// Start the firehose
func (f *Firehose) Start() {
	go f.sink.Start()

	// Stop chan for all tasks to depend on
	f.stopCh = make(chan struct{})

	// watch for allocation changes
	go f.watch()

	// Save the last event time every 5s
	go f.persistLastChangeTime(5 * time.Second)

	// wait forever for a stop signal to happen
	select {
	case <-f.stopCh:
		return
	}
}

// Stop the firehose
func (f *Firehose) Stop() {
	close(f.stopCh)
	f.sink.Stop()
}

// Write the Last Change Time to Consul so if the process restarts,
// it will try to resume from where it left off, not emitting tons of double events for
// old events
func (f *Firehose) persistLastChangeTime(interval time.Duration) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-f.stopCh:
			f.lastChangeTimeCh <- f.lastChangeIndex
			break
		case <-ticker.C:
			f.lastChangeTimeCh <- f.lastChangeIndex
		}
	}
}

// Publish an update from the firehose
func (f *Firehose) Publish(update *nomad.Evaluation) {
	b, err := json.Marshal(update)
	if err != nil {
		log.Error(err)
	}

	f.sink.Put(b)
}

// Continously watch for changes to the allocation list and publish it as updates
func (f *Firehose) watch() {
	q := &nomad.QueryOptions{
		WaitIndex:  f.lastChangeIndex,
		WaitTime:   5 * time.Minute,
		AllowStale: true,
	}

	newMax := f.lastChangeIndex

	for {
		log.Infof("Fetching evaluations from Nomad: %+v", q)

		evaluations, meta, err := f.nomadClient.Evaluations().List(q)
		if err != nil {
			log.Errorf("Unable to fetch evaluations: %s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		remoteWaitIndex := meta.LastIndex
		localWaitIndex := q.WaitIndex

		// Only work if the WaitIndex have changed
		if remoteWaitIndex == localWaitIndex {
			log.Infof("Evaluations index is unchanged (%d == %d)", meta.LastIndex, f.lastChangeIndex)
			continue
		}

		log.Infof("Evaluations index is changed (%d <> %d)", remoteWaitIndex, localWaitIndex)

		// Iterate clients and find events that have changed since last run
		for _, evaluation := range evaluations {
			if evaluation.ModifyIndex <= newMax {
				continue
			}

			if evaluation.ModifyIndex > newMax {
				newMax = evaluation.ModifyIndex
			}

			f.Publish(evaluation)
		}

		// Update WaitIndex and Last Change Time for next iteration
		f.lastChangeIndex = meta.LastIndex
		q.WaitIndex = meta.LastIndex
	}
}
