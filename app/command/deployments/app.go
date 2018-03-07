package deployments

import (
	"encoding/json"
	"os"
	"os/signal"
	"strconv"
	"time"

	"app/helper"
	"app/sink"

	consul "github.com/hashicorp/consul/api"
	nomad "github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
)

const (
	consulLockKey   = "nomad-firehose/deployments.lock"
	consulLockValue = "nomad-firehose/deployments.value"
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

	// watch for deployment changes
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
		WaitIndex:  f.lastChangeIndex,
		WaitTime:   5 * time.Minute,
		AllowStale: true,
	}

	newMax := f.lastChangeIndex

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
		if remoteWaitIndex <= localWaitIndex {
			log.Debugf("Deployments index is unchanged (%d <= %d)", remoteWaitIndex, localWaitIndex)
			continue
		}

		log.Debugf("Deployments index is changed (%d <> %d)", remoteWaitIndex, localWaitIndex)

		// Iterate deployments and find events that have changed since last run
		for _, deployment := range deployments {
			if deployment.ModifyIndex <= f.lastChangeIndex {
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
