package structs

import (
	"time"

	nomad "github.com/hashicorp/nomad/api"
)

// AllocationUpdate ...
type AllocationUpdate struct {
	Name               string
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
	TaskStartedAt      time.Time
	TaskFinishedAt     time.Time
	TaskEvent          *nomad.TaskEvent
}
