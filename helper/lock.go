package helper

import (
	"os"

	consul "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

// Wait for lock to the Consul KV key.
// This will ensure we are the only applicating running and processing
// allocation events to the firehose
func WaitForLock(key string) (*consul.Lock, string, error) {
	client, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return nil, "", err
	}

	log.Info("Trying to acquire leader lock")
	sessionID, err := session(client)
	if err != nil {
		return nil, "", err
	}

	prefix, ok := os.LookupEnv("CONSUL_LOCK_PREFIX")
	if !ok {
		prefix = "nomad-firehose/"
	}

	lock, err := client.LockOpts(&consul.LockOptions{
		Key:     prefix + key,
		Session: sessionID,
	})
	if err != nil {
		return nil, "", err
	}

	_, err = lock.Lock(nil)
	if err != nil {
		return nil, "", err
	}

	log.Info("Lock acquired")
	return lock, sessionID, nil
}

// Create a Consul session used for locks
func session(c *consul.Client) (string, error) {
	n, ok := os.LookupEnv("CONSUL_SESSION_NAME")
	if !ok {
		n = "nomad-firehose-allocations"
	}

	s := c.Session()
	se := &consul.SessionEntry{
		Name: n,
		TTL:  "15s",
	}

	id, _, err := s.Create(se, nil)
	if err != nil {
		return "", err
	}

	return id, nil
}
