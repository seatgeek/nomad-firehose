package allocations

import (
	consul "github.com/hashicorp/consul/api"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-firehose/structs"
)

// AllocationFirehose ...
type AllocationFirehose struct {
	nomadClient     *nomad.Client
	consulClient    *consul.Client
	consulSessionID string
	consulLock      *consul.Lock
	stopCh          chan struct{}
	lastChangeTime  int64
	sink            structs.Sink
}
