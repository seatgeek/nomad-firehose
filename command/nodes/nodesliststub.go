package nodes

import (
	"encoding/json"

	nomad "github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
)

// Firehose ...
type NodeListStubFirehose struct {
	FirehoseBase
}

// NewFirehose ...
func NewNodeListStubFirehose() (*NodeListStubFirehose, error) {
	base, err := NewFirehoseBase()
	if err != nil {
		return nil, err
	}

	return &NodeListStubFirehose{FirehoseBase: *base}, nil
}

func (f *NodeListStubFirehose) Name() string {
	return "nodeliststub"
}

func (f *NodeListStubFirehose) Start() {
	f.FirehoseBase.Start(f.watchNodeList)
}

// Publish an update from the firehose
func (f *NodeListStubFirehose) Publish(update *nomad.NodeListStub) {
	b, err := json.Marshal(update)
	if err != nil {
		log.Error(err)
	}

	f.sink.Put(b)
}

func (f *NodeListStubFirehose) watchNodeList(node *nomad.NodeListStub) {
	f.Publish(node)
}
