package helper

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

type Runner interface {
	Name() string
	SetRestoreValue(restoreTime interface{}) error
	Start()
	Stop()
	UpdateCh() <-chan interface{}
}

func NewManager(r Runner) *Manager {
	return &Manager{
		runner: r,
		logger: log.WithField("type", r.Name()),
		stopCh: make(chan interface{}),
		voluntarilyReleaseLockCh: make(chan interface{}),
	}
}

type Manager struct {
	runner                   Runner
	client                   *consulapi.Client
	lock                     *consulapi.Lock
	lockCh                   <-chan struct{}  // lock channel used by Consul SDK to notify about changes
	lockErrorCh              <-chan struct{}  // lock error channel used by Consul SDK to notify about errors related to the lock
	logger                   *log.Entry       // logger for the consul connection struct
	stopCh                   chan interface{} // internal channel used to stop all go-routines when gracefully shutting down
	voluntarilyReleaseLockCh chan interface{}
}

// cleanup will do cleanup tasks when the reconciler is shutting down
func (m *Manager) cleanup() {
	m.logger.Debug("Releasing lock")
	m.releaseConsulLock()

	m.logger.Debug("Closing stopCh")
	close(m.stopCh)

	m.logger.Debugf("Cleanup complete")
}

// continuouslyAcquireConsulLeadership waits to acquire the lock to the Consul KV key.
// it will run until the stopCh is closed
func (m *Manager) continuouslyAcquireConsulLeadership() error {
	m.logger.Info("Starting to continously acquire leadership")

	interval := 250 * time.Millisecond
	timer := time.NewTimer(interval)

	for {
		select {
		// if closed, we should stop working
		case <-m.stopCh:
			return nil

		// Periodically try to acquire the consul lock
		case <-timer.C:
			if err := m.acquireConsulLeadership(); err != nil {
				return err
			}
			timer.Reset(interval)
		}
	}
}

// Read the Last Change Time from Consul KV, so we don't re-process tasks over and over on restart
func (m *Manager) restoreLastChangeTime() interface{} {
	kv, _, err := m.client.KV().Get(fmt.Sprintf("nomad-firehose/%s.value", m.runner.Name()), nil)
	if err != nil {
		return 0
	}

	// Ensure we got
	if kv != nil && kv.Value != nil {
		sv := string(kv.Value)
		v, err := strconv.ParseInt(sv, 10, 64)
		if err != nil {
			return 0
		}

		log.Infof("Restoring Last Change Time to %s", sv)
		return v
	}

	log.Info("No Last Change Time restore point, starting from scratch")
	return 0
}

// acquireConsulLeadership will one-off try to acquire the consul lock needed to become
// redis Master node
func (m *Manager) acquireConsulLeadership() error {
	var err error
	m.lock, err = m.client.LockOpts(&consulapi.LockOptions{
		Key:              fmt.Sprintf("nomad-firehose/%s.lock", m.runner.Name()),
		SessionName:      fmt.Sprintf("nomad-firehose-%s", m.runner.Name()),
		MonitorRetries:   10,
		MonitorRetryTime: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("Failed create lock options: %+v", err)
	}

	// try to acquire the lock
	m.logger.Infof("Trying to acquire consul lock")
	m.lockErrorCh, err = m.lock.Lock(m.lockCh)
	if err != nil {
		return err
	}

	m.logger.Info("Lock successfully acquired")

	//
	// start monitoring the consul lock for errors / changes
	//

	m.voluntarilyReleaseLockCh = make(chan interface{})
	if err := m.runner.SetRestoreValue(m.restoreLastChangeTime()); err != nil {
		return err
	}

	go m.runner.Start()

	// At this point, if we return from this function, we need to make sure
	// we release the lock
	defer func() {
		m.runner.Stop()

		err := m.lock.Unlock()
		m.handleConsulError(err)
		if err != nil {
			m.logger.Errorf("Could not release Consul Lock: %v", err)
		} else {
			m.logger.Info("Consul Lock successfully released")
		}
	}()

	// Wait for changes to Consul Lock
	for {
		select {
		case v := <-m.runner.UpdateCh():
			var r string
			switch v.(type) {
			case int:
				r = strconv.Itoa(v.(int))
			case int64, uint64:
				r = fmt.Sprintf("%d", v)
			default:
				return fmt.Errorf("Unknown update type '%T' with value '%+v'", v, v)
			}

			m.logger.Debug("Writing lastChangedTime to KV: %s", r)
			kv := &consulapi.KVPair{
				Key:   fmt.Sprintf("nomad-firehose/%s.value", m.runner.Name()),
				Value: []byte(r),
			}
			_, err := m.client.KV().Put(kv, nil)
			if err != nil {
				log.Error(err)
			}

		// Global stop of all go-routines, reconciler is shutting down
		case <-m.stopCh:
			return nil

		// Changes on the lock error channel
		// if the channel is closed, it mean that we no longer hold the lock
		// if written to, we simply pass on the message
		case data, ok := <-m.lockErrorCh:
			if !ok {
				return fmt.Errorf("Consul Lock error channel was closed, we no longer hold the lock")
			}

			m.logger.Warnf("Something wrote to lock error channel %+v", data)

		// voluntarily release our claim on the lock
		case <-m.voluntarilyReleaseLockCh:
			m.logger.Warnf("Voluntarily releasing the Consul lock")
			return nil
		}
	}
}

// releaseConsulLock stops consul lock handler")
func (m *Manager) releaseConsulLock() {
	m.logger.Info("Releasing Consul lock")
	close(m.voluntarilyReleaseLockCh)
}

// handleConsulError is the error handler
func (m *Manager) handleConsulError(err error) {
	// if no error
	if err == nil {
		return
	}

	m.logger.Errorf("Consul error: %v", err)
}

func (m *Manager) Start() error {
	m.logger.Info("Starting manager")

	var err error
	m.client, err = consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		return err
	}

	go m.signalHandler()
	return m.continuouslyAcquireConsulLeadership()
}

// Close the stopCh if we get a signal, so we can gracefully shut down
func (m *Manager) signalHandler() {
	m.logger.Info("Starting signal handler")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		fmt.Println()
		log.Info("Caught signal, releasing lock and stopping...")
		m.cleanup()
	case <-m.stopCh:
		break
	}
}
