package health

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/AaronKaa/ok/internal/config"
)

type Notifier interface {
	Enabled() bool
	Continuous() bool
	Notify(status, previousStatus Status, failedChecks []Result)
}

type Service struct {
	checks         map[string]Check
	results        map[string]Result
	checkers       map[config.CheckType]Checker
	notifier       Notifier
	previousStatus Status
	mu             sync.RWMutex
}

func NewService(checks []Check, checkers map[config.CheckType]Checker, notifier Notifier) *Service {
	s := &Service{
		checks:         make(map[string]Check),
		results:        make(map[string]Result),
		checkers:       checkers,
		notifier:       notifier,
		previousStatus: StatusPass,
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

	checker, ok := s.checkers[check.Type]
	if !ok {
		slog.Error("no checker for check type", "check_id", checkID, "type", check.Type)
		return
	}

	start := time.Now()
	passed, message, err := checker.Execute(ctx, check)
	duration := time.Since(start)

	if err != nil {
		slog.Error("check execution error", "check_id", checkID, "error", err)
		message = err.Error()
		passed = false
	}

	s.mu.Lock()

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

	newStatus := s.aggregateStatusLocked()
	previousStatus := s.previousStatus
	statusChanged := newStatus != previousStatus
	s.previousStatus = newStatus

	var failedChecks []Result
	if newStatus != StatusPass {
		for _, r := range s.results {
			if r.Status != StatusPass {
				failedChecks = append(failedChecks, r)
			}
		}
	}

	s.mu.Unlock()

	slog.Info("check completed",
		"check_id", checkID,
		"status", result.Status,
		"duration", duration,
		"message", message,
	)

	s.maybeNotify(newStatus, previousStatus, statusChanged, failedChecks)
}

func (s *Service) maybeNotify(status, previousStatus Status, changed bool, failedChecks []Result) {
	if s.notifier == nil || !s.notifier.Enabled() {
		return
	}

	shouldNotify := false

	if changed {
		shouldNotify = true
	} else if s.notifier.Continuous() && status != StatusPass {
		shouldNotify = true
	}

	if shouldNotify {
		s.notifier.Notify(status, previousStatus, failedChecks)
	}
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
	return s.aggregateStatusLocked()
}

func (s *Service) aggregateStatusLocked() Status {
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
