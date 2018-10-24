package nodes

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
	lastChangeIndex   uint64
	lastChangeIndexCh chan interface{}
	nomadClient       *nomad.Client
	sink              sink.Sink
	stopCh            chan struct{}
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
		nomadClient:       nomadClient,
		sink:              sink,
		stopCh:            make(chan struct{}, 1),
		lastChangeIndexCh: make(chan interface{}, 1),
	}, nil
}

func (f *Firehose) Name() string {
	return "nodes"
}

func (f *Firehose) UpdateCh() <-chan interface{} {
	return f.lastChangeIndexCh
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
			f.lastChangeIndexCh <- f.lastChangeIndex
			break
		case <-ticker.C:
			f.lastChangeIndexCh <- f.lastChangeIndex
		}
	}
}

// Publish an update from the firehose
func (f *Firehose) Publish(update *nomad.Node) {
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
		clients, meta, err := f.nomadClient.Nodes().List(q)
		if err != nil {
			log.Errorf("Unable to fetch clients: %s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		remoteWaitIndex := meta.LastIndex
		localWaitIndex := q.WaitIndex

		// Only work if the WaitIndex have changed
		if remoteWaitIndex == localWaitIndex {
			log.Debugf("Clients index is unchanged (%d == %d)", remoteWaitIndex, localWaitIndex)
			continue
		}

		log.Debugf("Clients index is changed (%d <> %d)", remoteWaitIndex, localWaitIndex)

		// Iterate clients and find events that have changed since last run
		for _, client := range clients {
			if client.ModifyIndex < newMax {
				continue
			}

			if client.ModifyIndex > newMax {
				newMax = client.ModifyIndex
			}

			go func(clientId string) {
				fullClient, _, err := f.nomadClient.Nodes().Info(clientId, &nomad.QueryOptions{})
				if err != nil {
					log.Errorf("Could not read client %s: %s", clientId, err)
					return
				}

				f.Publish(fullClient)
			}(client.ID)
		}

		// Update WaitIndex and Last Change Time for next iteration
		q.WaitIndex = meta.LastIndex
		f.lastChangeIndex = newMax
	}
}
