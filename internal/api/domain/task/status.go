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

// Git push/PR status values stored on Task.GitPushStatus and Task.GitPRStatus.
const (
	GitStatusOK      = "ok"
	GitStatusError   = "error"
	GitStatusSkipped = "skipped"
)

func (s ExecutionStatus) Valid() bool {
	switch s {
	case StatusIdle, StatusQueued, StatusRunning, StatusSucceeded, StatusFailed, StatusDone:
		return true
	default:
		return false
	}
}
