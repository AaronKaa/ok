package health

import (
	"context"
	"testing"

	"github.com/AaronKaa/ok/internal/config"
)

type mockChecker struct {
	passed  bool
	message string
	err     error
}

func (m *mockChecker) Execute(ctx context.Context, check Check) (bool, string, error) {
	return m.passed, m.message, m.err
}

func mockCheckers(checker Checker) map[config.CheckType]Checker {
	return map[config.CheckType]Checker{
		config.CheckTypeHTTP: checker,
	}
}

type mockNotifier struct {
	enabled      bool
	continuous   bool
	notifyCalls  int
	lastStatus   Status
	lastPrevious Status
	lastFailed   []Result
}

func (m *mockNotifier) Enabled() bool    { return m.enabled }
func (m *mockNotifier) Continuous() bool { return m.continuous }
func (m *mockNotifier) Notify(status, previousStatus Status, failedChecks []Result) {
	m.notifyCalls++
	m.lastStatus = status
	m.lastPrevious = previousStatus
	m.lastFailed = failedChecks
}

func TestService_Execute(t *testing.T) {
	checks := []Check{
		{ID: "test1", Type: config.CheckTypeHTTP, Title: "Test 1", Critical: true, Retries: 0},
		{ID: "test2", Type: config.CheckTypeHTTP, Title: "Test 2", Critical: false, Retries: 1},
	}

	t.Run("successful check", func(t *testing.T) {
		checker := &mockChecker{passed: true, message: "ok"}
		svc := NewService(checks, mockCheckers(checker), nil)

		svc.Execute(context.Background(), "test1")

		result, ok := svc.Result("test1")
		if !ok {
			t.Fatal("result not found")
		}
		if result.Status != StatusPass {
			t.Errorf("Status = %v, want %v", result.Status, StatusPass)
		}
		if result.ConsecutiveFailures != 0 {
			t.Errorf("ConsecutiveFailures = %v, want 0", result.ConsecutiveFailures)
		}
	})

	t.Run("failed critical check", func(t *testing.T) {
		checker := &mockChecker{passed: false, message: "error"}
		svc := NewService(checks, mockCheckers(checker), nil)

		svc.Execute(context.Background(), "test1")

		result, _ := svc.Result("test1")
		if result.Status != StatusFail {
			t.Errorf("Status = %v, want %v", result.Status, StatusFail)
		}
		if result.ConsecutiveFailures != 1 {
			t.Errorf("ConsecutiveFailures = %v, want 1", result.ConsecutiveFailures)
		}
	})

	t.Run("failed non-critical check", func(t *testing.T) {
		checker := &mockChecker{passed: false, message: "error"}
		svc := NewService(checks, mockCheckers(checker), nil)

		svc.Execute(context.Background(), "test2")

		result, _ := svc.Result("test2")
		if result.Status != StatusDegraded {
			t.Errorf("Status = %v, want %v", result.Status, StatusDegraded)
		}
	})

	t.Run("recovery resets failures", func(t *testing.T) {
		checker := &mockChecker{passed: false, message: "error"}
		svc := NewService(checks, mockCheckers(checker), nil)

		svc.Execute(context.Background(), "test1")
		svc.Execute(context.Background(), "test1")

		result, _ := svc.Result("test1")
		if result.ConsecutiveFailures != 2 {
			t.Errorf("ConsecutiveFailures = %v, want 2", result.ConsecutiveFailures)
		}

		checker.passed = true
		svc.Execute(context.Background(), "test1")

		result, _ = svc.Result("test1")
		if result.ConsecutiveFailures != 0 {
			t.Errorf("ConsecutiveFailures after recovery = %v, want 0", result.ConsecutiveFailures)
		}
		if result.Status != StatusPass {
			t.Errorf("Status after recovery = %v, want %v", result.Status, StatusPass)
		}
	})
}

func TestService_AggregateStatus(t *testing.T) {
	checks := []Check{
		{ID: "critical", Type: config.CheckTypeHTTP, Title: "Critical", Critical: true},
		{ID: "non-critical", Type: config.CheckTypeHTTP, Title: "Non-Critical", Critical: false},
	}

	t.Run("all pass", func(t *testing.T) {
		checker := &mockChecker{passed: true}
		svc := NewService(checks, mockCheckers(checker), nil)

		svc.Execute(context.Background(), "critical")
		svc.Execute(context.Background(), "non-critical")

		if status := svc.AggregateStatus(); status != StatusPass {
			t.Errorf("AggregateStatus() = %v, want %v", status, StatusPass)
		}
	})

	t.Run("critical fails", func(t *testing.T) {
		checker := &mockChecker{passed: true}
		svc := NewService(checks, mockCheckers(checker), nil)
		svc.Execute(context.Background(), "non-critical")

		checker.passed = false
		svc.Execute(context.Background(), "critical")

		if status := svc.AggregateStatus(); status != StatusFail {
			t.Errorf("AggregateStatus() = %v, want %v", status, StatusFail)
		}
	})

	t.Run("non-critical fails only", func(t *testing.T) {
		checker := &mockChecker{passed: true}
		svc := NewService(checks, mockCheckers(checker), nil)
		svc.Execute(context.Background(), "critical")

		checker.passed = false
		svc.Execute(context.Background(), "non-critical")

		if status := svc.AggregateStatus(); status != StatusDegraded {
			t.Errorf("AggregateStatus() = %v, want %v", status, StatusDegraded)
		}
	})
}

func TestService_Results(t *testing.T) {
	checks := []Check{
		{ID: "a", Type: config.CheckTypeHTTP, Title: "A"},
		{ID: "b", Type: config.CheckTypeHTTP, Title: "B"},
	}

	svc := NewService(checks, mockCheckers(&mockChecker{passed: true}), nil)
	results := svc.Results()

	if len(results) != 2 {
		t.Errorf("Results() len = %v, want 2", len(results))
	}
}

func TestService_Checks(t *testing.T) {
	checks := []Check{
		{ID: "a", Type: config.CheckTypeHTTP, Title: "A"},
		{ID: "b", Type: config.CheckTypeHTTP, Title: "B"},
	}

	svc := NewService(checks, mockCheckers(&mockChecker{}), nil)
	got := svc.Checks()

	if len(got) != 2 {
		t.Errorf("Checks() len = %v, want 2", len(got))
	}
}

func TestService_Notifications(t *testing.T) {
	checks := []Check{
		{ID: "test", Type: config.CheckTypeHTTP, Title: "Test", Critical: true},
	}

	t.Run("notifies on state change to fail", func(t *testing.T) {
		notifier := &mockNotifier{enabled: true}
		checker := &mockChecker{passed: false}
		svc := NewService(checks, mockCheckers(checker), notifier)

		svc.Execute(context.Background(), "test")

		if notifier.notifyCalls != 1 {
			t.Errorf("notifyCalls = %v, want 1", notifier.notifyCalls)
		}
		if notifier.lastStatus != StatusFail {
			t.Errorf("lastStatus = %v, want %v", notifier.lastStatus, StatusFail)
		}
		if notifier.lastPrevious != StatusPass {
			t.Errorf("lastPrevious = %v, want %v", notifier.lastPrevious, StatusPass)
		}
		if len(notifier.lastFailed) != 1 {
			t.Errorf("lastFailed len = %v, want 1", len(notifier.lastFailed))
		}
	})

	t.Run("notifies on state change to pass", func(t *testing.T) {
		notifier := &mockNotifier{enabled: true}
		checker := &mockChecker{passed: false}
		svc := NewService(checks, mockCheckers(checker), notifier)

		svc.Execute(context.Background(), "test") // fail
		checker.passed = true
		svc.Execute(context.Background(), "test") // pass

		if notifier.notifyCalls != 2 {
			t.Errorf("notifyCalls = %v, want 2", notifier.notifyCalls)
		}
		if notifier.lastStatus != StatusPass {
			t.Errorf("lastStatus = %v, want %v", notifier.lastStatus, StatusPass)
		}
	})

	t.Run("does not notify when no state change", func(t *testing.T) {
		notifier := &mockNotifier{enabled: true}
		checker := &mockChecker{passed: false}
		svc := NewService(checks, mockCheckers(checker), notifier)

		svc.Execute(context.Background(), "test") // fail (notifies)
		svc.Execute(context.Background(), "test") // still fail (no notify)

		if notifier.notifyCalls != 1 {
			t.Errorf("notifyCalls = %v, want 1", notifier.notifyCalls)
		}
	})

	t.Run("continuous mode notifies on every failure", func(t *testing.T) {
		notifier := &mockNotifier{enabled: true, continuous: true}
		checker := &mockChecker{passed: false}
		svc := NewService(checks, mockCheckers(checker), notifier)

		svc.Execute(context.Background(), "test") // fail
		svc.Execute(context.Background(), "test") // still fail

		if notifier.notifyCalls != 2 {
			t.Errorf("notifyCalls = %v, want 2", notifier.notifyCalls)
		}
	})

	t.Run("disabled notifier does not notify", func(t *testing.T) {
		notifier := &mockNotifier{enabled: false}
		checker := &mockChecker{passed: false}
		svc := NewService(checks, mockCheckers(checker), notifier)

		svc.Execute(context.Background(), "test")

		if notifier.notifyCalls != 0 {
			t.Errorf("notifyCalls = %v, want 0", notifier.notifyCalls)
		}
	})
}
