package helper

import (
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
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

	lock, err := client.LockOpts(&consul.LockOptions{
		Key:     key,
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
	s := c.Session()
	se := &consul.SessionEntry{
		Name: "nomad-firehose-allocations",
		TTL:  "15s",
	}

	id, _, err := s.Create(se, nil)
	if err != nil {
		return "", err
	}

	return id, nil
}
