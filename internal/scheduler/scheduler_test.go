package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AaronKaa/ok/internal/health"
)

type mockService struct {
	checks       []health.Check
	executeCalls atomic.Int32
	mu           sync.Mutex
	executed     []string
}

func (m *mockService) Checks() []health.Check {
	return m.checks
}

func (m *mockService) Execute(ctx context.Context, checkID string) {
	m.executeCalls.Add(1)
	m.mu.Lock()
	m.executed = append(m.executed, checkID)
	m.mu.Unlock()
}

func (m *mockService) ExecutedIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.executed...)
}

func TestScheduler_Start(t *testing.T) {
	svc := &mockService{
		checks: []health.Check{
			{ID: "test1", Interval: 50 * time.Millisecond},
		},
	}

	sched := New(svc)
	sched.Start(context.Background())

	time.Sleep(80 * time.Millisecond)
	sched.Stop()

	calls := int(svc.executeCalls.Load())
	if calls < 1 {
		t.Errorf("Execute() called %d times, want >= 1", calls)
	}
}

func TestScheduler_MultipleChecks(t *testing.T) {
	svc := &mockService{
		checks: []health.Check{
			{ID: "check1", Interval: 50 * time.Millisecond},
			{ID: "check2", Interval: 50 * time.Millisecond},
		},
	}

	sched := New(svc)
	sched.Start(context.Background())

	time.Sleep(30 * time.Millisecond)
	sched.Stop()

	executed := svc.ExecutedIDs()
	hasCheck1, hasCheck2 := false, false
	for _, id := range executed {
		if id == "check1" {
			hasCheck1 = true
		}
		if id == "check2" {
			hasCheck2 = true
		}
	}

	if !hasCheck1 || !hasCheck2 {
		t.Errorf("Expected both checks to be executed, got: %v", executed)
	}
}

func TestScheduler_Stop(t *testing.T) {
	svc := &mockService{
		checks: []health.Check{
			{ID: "test", Interval: 10 * time.Millisecond},
		},
	}

	sched := New(svc)
	sched.Start(context.Background())

	time.Sleep(25 * time.Millisecond)
	sched.Stop()

	callsAtStop := svc.executeCalls.Load()
	time.Sleep(30 * time.Millisecond)
	callsAfter := svc.executeCalls.Load()

	if callsAfter != callsAtStop {
		t.Errorf("Scheduler continued after Stop(): calls went from %d to %d", callsAtStop, callsAfter)
	}
}

func TestScheduler_ImmediateExecution(t *testing.T) {
	svc := &mockService{
		checks: []health.Check{
			{ID: "test", Interval: 1 * time.Hour},
		},
	}

	sched := New(svc)
	sched.Start(context.Background())

	time.Sleep(20 * time.Millisecond)
	sched.Stop()

	if calls := svc.executeCalls.Load(); calls != 1 {
		t.Errorf("Execute() called %d times, want 1 (immediate execution)", calls)
	}
}
