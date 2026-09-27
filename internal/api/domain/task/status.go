package task

type ExecutionStatus string

const (
	StatusIdle      ExecutionStatus = "idle"
	StatusQueued    ExecutionStatus = "queued"
	StatusRunning   ExecutionStatus = "running"
	StatusSucceeded ExecutionStatus = "succeeded"
	StatusFailed    ExecutionStatus = "failed"
	StatusDone      ExecutionStatus = "done"
)

func (s ExecutionStatus) Valid() bool {
	switch s {
	case StatusIdle, StatusQueued, StatusRunning, StatusSucceeded, StatusFailed, StatusDone:
		return true
	default:
		return false
	}
}
