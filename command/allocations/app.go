package allocations

import (
	"encoding/json"
	"fmt"
	"time"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-firehose/sink"
	log "github.com/sirupsen/logrus"
)

// Firehose ...
type Firehose struct {
	lastChangeTime   int64
	lastChangeTimeCh chan interface{}
	nomadClient      *nomad.Client
	sink             sink.Sink
	stopCh           chan struct{}
}

// AllocationUpdate ...
type AllocationUpdate struct {
	Name               string
	NodeID             string
	AllocationID       string
	DesiredStatus      string
	DesiredDescription string
	ClientStatus       string
	ClientDescription  string
	JobID              string
	GroupName          string
	TaskName           string
	EvalID             string
	TaskState          string
	TaskFailed         bool
	TaskStartedAt      *time.Time
	TaskFinishedAt     *time.Time
	TaskEvent          *nomad.TaskEvent
	ModifyTime		   int64
}

// NewFirehose ...
func NewFirehose() (*Firehose, error) {
	nomadClient, err := nomad.NewClient(nomad.DefaultConfig())
	if err != nil {
		return nil, err
	}

	sink, err := sink.GetSink()
	if err != nil {
		return nil, err
	}

	return &Firehose{
		nomadClient:      nomadClient,
		sink:             sink,
		stopCh:           make(chan struct{}, 1),
		lastChangeTimeCh: make(chan interface{}, 1),
	}, nil
}

func (f *Firehose) Name() string {
	return "allocations"
}

func (f *Firehose) UpdateCh() <-chan interface{} {
	return f.lastChangeTimeCh
}

func (f *Firehose) SetRestoreValue(restoreValue interface{}) error {
	switch restoreValue.(type) {
	case int:
		f.lastChangeTime = int64(restoreValue.(int))
	case int64:
		f.lastChangeTime = restoreValue.(int64)
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

// publish an update from the firehose
func (f *Firehose) publish(update *AllocationUpdate) {
	b, err := json.Marshal(update)
	if err != nil {
		log.Error(err)
	}

	f.sink.Put(update.AllocationID, b)
}

// Continously watch for changes to the allocation list and publish it as updates
func (f *Firehose) watch() {
	q := &nomad.QueryOptions{
		WaitIndex:  1,
		WaitTime:   5 * time.Minute,
		AllowStale: true,
	}

	newMax := f.lastChangeTime

	for {
		allocations, meta, err := f.nomadClient.Allocations().List(q)
		if err != nil {
			log.Errorf("Unable to fetch allocations: %s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		remoteWaitIndex := meta.LastIndex
		localWaitIndex := q.WaitIndex

		// Only work if the WaitIndex have changed
		if remoteWaitIndex == localWaitIndex {
			log.Debugf("Allocations index is unchanged (%d == %d)", remoteWaitIndex, localWaitIndex)
			continue
		}

		log.Debugf("Allocations index is changed (%d <> %d)", remoteWaitIndex, localWaitIndex)

		// Iterate allocations and find events that have changed since last run
		for _, allocation := range allocations {
			has_published := false
			for taskName, taskInfo := range allocation.TaskStates {
				for _, taskEvent := range taskInfo.Events {
					if taskEvent.Time <= f.lastChangeTime {
						continue
					}

					if taskEvent.Time > newMax {
						newMax = taskEvent.Time
					}

					payload := &AllocationUpdate{
						Name:               allocation.Name,
						NodeID:             allocation.NodeID,
						AllocationID:       allocation.ID,
						EvalID:             allocation.EvalID,
						DesiredStatus:      allocation.DesiredStatus,
						DesiredDescription: allocation.DesiredDescription,
						ClientStatus:       allocation.ClientStatus,
						ClientDescription:  allocation.ClientDescription,
						JobID:              allocation.JobID,
						GroupName:          allocation.TaskGroup,
						TaskName:           taskName,
						TaskEvent:          taskEvent,
						TaskState:          taskInfo.State,
						TaskFailed:         taskInfo.Failed,
						TaskStartedAt:      &taskInfo.StartedAt,
						TaskFinishedAt:     &taskInfo.FinishedAt,
						ModifyTime:         allocation.ModifyTime
					}

					f.publish(payload)
					has_published = true
				}

				if !has_published && allocation.ModifyTime >= f.lastChangeTime {
					payload := &AllocationUpdate{
						Name:               allocation.Name,
						NodeID:             allocation.NodeID,
						AllocationID:       allocation.ID,
						EvalID:             allocation.EvalID,
						DesiredStatus:      allocation.DesiredStatus,
						DesiredDescription: allocation.DesiredDescription,
						ClientStatus:       allocation.ClientStatus,
						ClientDescription:  allocation.ClientDescription,
						JobID:              allocation.JobID,
						GroupName:          allocation.TaskGroup,
						TaskName:           taskName,
						TaskEvent:          nil,
						TaskState:          taskInfo.State,
						TaskFailed:         taskInfo.Failed,
						TaskStartedAt:      &taskInfo.StartedAt,
						TaskFinishedAt:     &taskInfo.FinishedAt,
						ModifyTime:         allocation.ModifyTime,
					}

					f.publish(payload)
					has_published = true
				}
			}
		}

		// Update WaitIndex and Last Change Time for next iteration
		q.WaitIndex = meta.LastIndex
		f.lastChangeTime = newMax
	}
}
