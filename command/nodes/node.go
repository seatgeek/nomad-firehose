package nodes

import (
	"encoding/json"

	nomad "github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
)

// Firehose ...
type NodeFirehose struct {
	FirehoseBase
}

// NewFirehose ...
func NewNodeFirehose() (*NodeFirehose, error) {
	base, err := NewFirehoseBase()
	if err != nil {
		return nil, err
	}

	return &NodeFirehose{FirehoseBase: *base}, nil
}

func (f *NodeFirehose) Name() string {
	return "nodeliststub"
}

func (f *NodeFirehose) Start() {
	f.FirehoseBase.Start(f.watchNodeList)
}

// Publish an update from the firehose
func (f *NodeFirehose) Publish(update *nomad.Node) {
	b, err := json.Marshal(update)
	if err != nil {
		log.Error(err)
	}

	f.sink.Put(b)
}


func (f *NodeFirehose) watchNodeList(node *nomad.NodeListStub) {
	go func(nodeID string) {
		fullNode, _, err := f.nomadClient.Nodes().Info(nodeID, &nomad.QueryOptions{})
		if err != nil {
			log.Errorf("Could not read job %s: %s", nodeID, err)
			return
		}

		f.Publish(fullNode)
	}(node.ID)
}
