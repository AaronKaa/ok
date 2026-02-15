package health

type Status string

const (
	StatusPass     Status = "pass"
	StatusDegraded Status = "degraded"
	StatusFail     Status = "fail"
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsHealthy() bool {
	return s == StatusPass || s == StatusDegraded
}
