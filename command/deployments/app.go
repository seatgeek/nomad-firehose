package deployments

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-firehose/sink"
	log "github.com/sirupsen/logrus"
)

// Firehose ...
type Firehose struct {
	lastChangeTime   uint64
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
		lastChangeTimeCh: make(chan interface{}, 1),
	}, nil
}

func (f *Firehose) Name() string {
	return "deployments"
}

func (f *Firehose) UpdateCh() <-chan interface{} {
	return f.lastChangeTimeCh
}

func (f *Firehose) SetRestoreValue(restoreValue interface{}) error {
	switch restoreValue.(type) {
	case int:
		f.lastChangeTime = uint64(restoreValue.(int))
	case int64:
		f.lastChangeTime = uint64(restoreValue.(int64))
	case string:
		restoreValueInt, _ := strconv.Atoi(restoreValue.(string))
		f.lastChangeTime = uint64(restoreValueInt)
	default:
		return fmt.Errorf("Unable to compute restore time, not int or string (%T)", restoreValue)
	}

	return nil
}

// Start the firehose
func (f *Firehose) Start() {
	go f.sink.Start()

	// Stop chan for all tasks to depend on
	f.stopCh = make(chan struct{})

	// watch for deployment changes
	go f.watch()

	// Save the last event time every 5s
	go f.persistLastChangeTime(5 * time.Second)

	// wait forever for a stop signal to happen
	for {
		select {
		case <-f.stopCh:
			return
		}
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
			f.lastChangeTimeCh <- f.lastChangeTime
			break
		case <-ticker.C:
			f.lastChangeTimeCh <- f.lastChangeTime
		}
	}
}

// Publish an update from the firehose
func (f *Firehose) Publish(update *nomad.Deployment) {
	b, err := json.Marshal(update)
	if err != nil {
		log.Error(err)
	}

	f.sink.Put(b)
}

// Continously watch for changes to the deployment list and publish it as updates
func (f *Firehose) watch() {
	q := &nomad.QueryOptions{
		WaitIndex:  uint64(f.lastChangeTime),
		WaitTime:   5 * time.Minute,
		AllowStale: true,
	}

	newMax := uint64(f.lastChangeTime)

	for {
		deployments, meta, err := f.nomadClient.Deployments().List(q)
		if err != nil {
			log.Errorf("Unable to fetch deployments: %s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		remoteWaitIndex := meta.LastIndex
		localWaitIndex := q.WaitIndex

		// Only work if the WaitIndex have changed
		if remoteWaitIndex == localWaitIndex {
			log.Debugf("Deployments index is unchanged (%d == %d)", remoteWaitIndex, localWaitIndex)
			continue
		}

		log.Debugf("Deployments index is changed (%d <> %d)", remoteWaitIndex, localWaitIndex)

		// Iterate deployments and find events that have changed since last run
		for _, deployment := range deployments {
			if deployment.ModifyIndex <= newMax {
				continue
			}

			if deployment.ModifyIndex > newMax {
				newMax = deployment.ModifyIndex
			}

			go func(DeploymentID string) {
				fullDeployment, _, err := f.nomadClient.Deployments().Info(DeploymentID, &nomad.QueryOptions{})
				if err != nil {
					log.Errorf("Could not read deployment %s: %s", DeploymentID, err)
					return
				}

				f.Publish(fullDeployment)
			}(deployment.ID)
		}

		// Update WaitIndex and Last Change Time for next iteration
		q.WaitIndex = meta.LastIndex
		f.lastChangeTime = newMax
	}
}
