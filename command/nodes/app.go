package nodes

import (
	"encoding/json"
	"os"
	"os/signal"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-firehose/helper"
	"github.com/seatgeek/nomad-firehose/sink"
)

const (
	consulLockKey   = "nomad-firehose/clients.lock"
	consulLockValue = "nomad-firehose/clients.value"
)

// Firehose ...
type Firehose struct {
	nomadClient     *nomad.Client
	consulClient    *consul.Client
	consulSessionID string
	consulLock      *consul.Lock
	stopCh          chan struct{}
	lastChangeIndex uint64
	sink            sink.Sink
}

// NewFirehose ...
func NewFirehose() (*Firehose, error) {
	lock, sessionID, err := helper.WaitForLock(consulLockKey)
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	nomadClient, err := nomad.NewClient(nomad.DefaultConfig())
	if err != nil {
		return nil, err
	}

	consulClient, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return nil, err
	}

	sink, err := sink.GetSink()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	return &Firehose{
		nomadClient:     nomadClient,
		consulClient:    consulClient,
		consulSessionID: sessionID,
		consulLock:      lock,
		sink:            sink,
	}, nil
}

// Start the firehose
func (f *Firehose) Start() error {
	// Restore the last change time from Consul
	err := f.restoreLastChangeTime()
	if err != nil {
		return err
	}

	go f.sink.Start()

	// Stop chan for all tasks to depend on
	f.stopCh = make(chan struct{})

	// setup signal handler for graceful shutdown
	go f.signalHandler()

	// watch for allocation changes
	go f.watch()

	// Save the last event time every 10s
	go f.persistLastChangeTime(10)

	// wait forever for a stop signal to happen
	for {
		select {
		case <-f.stopCh:
			return nil
		}
	}
}

// Stop the firehose
func (f *Firehose) Stop() {
	close(f.stopCh)
	f.sink.Stop()
	f.writeLastChangeTime()
}

// Read the Last Change Time from Consul KV, so we don't re-process tasks over and over on restart
func (f *Firehose) restoreLastChangeTime() error {
	kv, _, err := f.consulClient.KV().Get(consulLockValue, &consul.QueryOptions{})
	if err != nil {
		return err
	}

	// Ensure we got
	if kv != nil && kv.Value != nil {
		sv := string(kv.Value)
		v, err := strconv.ParseInt(sv, 10, 64)
		if err != nil {
			return err
		}

		f.lastChangeIndex = uint64(v)
		log.Infof("Restoring Last Change Time to %s", sv)
	} else {
		log.Info("No Last Change Time restore point, starting from scratch")
	}

	return nil
}

// Write the Last Change Time to Consul so if the process restarts,
// it will try to resume from where it left off, not emitting tons of double events for
// old events
func (f *Firehose) persistLastChangeTime(interval time.Duration) {
	ticker := time.NewTicker(interval * time.Second)

	for {
		select {
		case <-f.stopCh:
			break
		case <-ticker.C:
			f.writeLastChangeTime()
		}
	}
}

func (f *Firehose) writeLastChangeTime() {
	v := strconv.FormatUint(f.lastChangeIndex, 10)

	log.Infof("Writing lastChangedTime to KV: %s", v)
	kv := &consul.KVPair{
		Key:     consulLockValue,
		Value:   []byte(v),
		Session: f.consulSessionID,
	}
	_, err := f.consulClient.KV().Put(kv, &consul.WriteOptions{})
	if err != nil {
		log.Error(err)
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
	q := &nomad.QueryOptions{WaitIndex: 1, AllowStale: true}

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
		if remoteWaitIndex <= localWaitIndex {
			log.Debugf("Clients index is unchanged (%d <= %d)", remoteWaitIndex, localWaitIndex)
			continue
		}

		log.Debugf("Clients index is changed (%d <> %d)", remoteWaitIndex, localWaitIndex)

		// Iterate clients and find events that have changed since last run
		for _, client := range clients {
			if client.ModifyIndex <= f.lastChangeIndex {
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

// Close the stopCh if we get a signal, so we can gracefully shut down
func (f *Firehose) signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		log.Info("Caught signal, releasing lock and stopping...")
		f.Stop()
	case <-f.stopCh:
		break
	}
}
