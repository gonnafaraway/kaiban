package task

import "errors"

var (
	ErrInvalidTransition = errors.New("invalid task transition")
	ErrCommentRequired   = errors.New("comment is required")
	ErrNotSucceeded      = errors.New("task is not succeeded")
	ErrAlreadyRunning    = errors.New("task is already running")
	ErrForwardMove       = errors.New("cannot move task forward without approve")
	ErrReturnTarget      = errors.New("return target must be a previous column")
	ErrTitleRequired     = errors.New("title is required")
	ErrArchived          = errors.New("task is archived")
	ErrAlreadyArchived   = errors.New("task is already archived")
	ErrNotArchived       = errors.New("task is not archived")
	ErrContractInvalid   = errors.New("stage contract is not satisfied")
	ErrGitDiffRequired   = errors.New("nonempty git diff is required")
	ErrBudgetExceeded    = errors.New("agent run budget exceeded")
)
