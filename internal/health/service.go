package health

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Service struct {
	checks  map[string]Check
	results map[string]Result
	checker Checker
	mu      sync.RWMutex
}

func NewService(checks []Check, checker Checker) *Service {
	s := &Service{
		checks:  make(map[string]Check),
		results: make(map[string]Result),
		checker: checker,
	}

	for _, check := range checks {
		s.checks[check.ID] = check
		s.results[check.ID] = NewResult(check)
	}

	return s
}

func (s *Service) Checks() []Check {
	s.mu.RLock()
	defer s.mu.RUnlock()

	checks := make([]Check, 0, len(s.checks))
	for _, check := range s.checks {
		checks = append(checks, check)
	}
	return checks
}

func (s *Service) Execute(ctx context.Context, checkID string) {
	s.mu.RLock()
	check, ok := s.checks[checkID]
	currentResult := s.results[checkID]
	s.mu.RUnlock()

	if !ok {
		slog.Error("check not found", "check_id", checkID)
		return
	}

	start := time.Now()
	passed, message, err := s.checker.Execute(ctx, check)
	duration := time.Since(start)

	if err != nil {
		slog.Error("check execution error", "check_id", checkID, "error", err)
		message = err.Error()
		passed = false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := s.results[checkID]
	result.LastCheck = time.Now()
	result.Duration = duration
	result.Message = message

	if passed {
		result.ConsecutiveFailures = 0
		result.Status = StatusPass
	} else {
		result.ConsecutiveFailures = currentResult.ConsecutiveFailures + 1
		result.Status = result.ComputeStatus(false)
	}

	s.results[checkID] = result

	slog.Info("check completed",
		"check_id", checkID,
		"status", result.Status,
		"duration", duration,
		"message", message,
	)
}

func (s *Service) Results() []Result {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]Result, 0, len(s.results))
	for _, result := range s.results {
		results = append(results, result)
	}
	return results
}

func (s *Service) Result(checkID string) (Result, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, ok := s.results[checkID]
	return result, ok
}

func (s *Service) AggregateStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hasDegraded := false

	for _, result := range s.results {
		switch result.Status {
		case StatusFail:
			return StatusFail
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasDegraded {
		return StatusDegraded
	}
	return StatusPass
}
