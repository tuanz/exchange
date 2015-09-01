package result

import (
	"time"
)

type Result struct {
	// This is set to true when the process that produces
	// this result starts.
	Attempted      bool

	// AttemptNumber is the number of the read attempt.
	// This starts at one.
	AttemptNumber  int

	// Errors is a list of strings describing errors that occurred
	// during bag validation.
	Errors         []string

	// StartedAt describes when the attempt to read the bag started.
	// If StartedAt.IsZero(), we have not yet attempted to read the
	// bag.
	StartedAt      time.Time

	// CompletedAt describes when the attempt to read the bag completed.
	// If CompletedAt.IsZero(), we have not yet attempted to read the
	// bag. Note that the attempt may have completed without succeeding.
	// Check the Succeeded() method to see if the process actually
	// completed successfully.
	CompletedAt    time.Time

	// Retry indicates whether we should retry a failed process.
	// After non-fatal errors, such as network timeout, this will
	// generally be set to true. For fatal errors, such as invalid
	// data, this will generally be set to false. This defaults to
	// true, because fatal errors are rare, and we don't want to
	// give up on transient errors. Just requeue and try again.
	Retry          bool
}

func NewResult() Result {
	return Result{
		Attempted: false,
		AttemptNumber: 1,
		Errors: make([]string, 0),
		StartedAt: time.Time{},
		CompletedAt: time.Time{},
		Retry: true,
	}
}

func (result *Result) Start() {
	result.StartedAt = time.Now()
}

func (result *Result) Started() bool {
	return !result.StartedAt.IsZero()
}

func (result *Result) Finish()  {
	result.CompletedAt = time.Now()
}

func (result *Result) Completed() bool {
	return !result.CompletedAt.IsZero()
}

func (result *Result) RunTime() time.Duration {
	startTime := result.StartedAt
	if startTime.IsZero() {
		return time.Duration(0)
	}
	endTime := result.CompletedAt
	if endTime.IsZero() {
		endTime = time.Now()
	}
	return endTime.Sub(startTime)
}

func (result *Result) Succeeded() bool {
	return result.Completed() && len(result.Errors) == 0
}

func (result *Result) AddError(errStr string) {
	result.Errors = append(result.Errors, errStr)
}
