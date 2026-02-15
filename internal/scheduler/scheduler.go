package scheduler

import (
	"context"
	"log/slog"
	"sync"

	"github.com/AaronKaa/ok/internal/health"
)

type Service interface {
	Checks() []health.Check
	Execute(ctx context.Context, checkID string)
}

type Scheduler struct {
	service Service
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

func New(service Service) *Scheduler {
	return &Scheduler{
		service: service,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)

	checks := s.service.Checks()
	slog.Info("starting scheduler", "check_count", len(checks))

	for _, check := range checks {
		s.wg.Add(1)
		go s.runCheck(ctx, check)
	}
}

func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	slog.Info("scheduler stopped")
}

func (s *Scheduler) runCheck(ctx context.Context, check health.Check) {
	defer s.wg.Done()

	slog.Info("starting check", "check_id", check.ID, "interval", check.Interval)

	s.service.Execute(ctx, check.ID)

	ticker := NewTicker(check.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			s.service.Execute(ctx, check.ID)
		}
	}
}
