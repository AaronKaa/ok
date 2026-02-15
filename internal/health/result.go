package health

import "time"

type Result struct {
	CheckID             string
	Title               string
	Status              Status
	Critical            bool
	LastCheck           time.Time
	Duration            time.Duration
	Message             string
	ConsecutiveFailures int
	Retries             int
}

func NewResult(check Check) Result {
	return Result{
		CheckID:  check.ID,
		Title:    check.Title,
		Status:   StatusPass,
		Critical: check.Critical,
		Retries:  check.Retries,
	}
}

func (r Result) Age() time.Duration {
	if r.LastCheck.IsZero() {
		return 0
	}
	return time.Since(r.LastCheck)
}

func (r Result) ComputeStatus(passed bool) Status {
	if passed {
		return StatusPass
	}

	failures := r.ConsecutiveFailures + 1

	if failures <= r.Retries {
		return StatusDegraded
	}

	if r.Critical {
		return StatusFail
	}
	return StatusDegraded
}
