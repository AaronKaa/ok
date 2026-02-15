package health

import (
	"context"
	"testing"
)

type mockChecker struct {
	passed  bool
	message string
	err     error
}

func (m *mockChecker) Execute(ctx context.Context, check Check) (bool, string, error) {
	return m.passed, m.message, m.err
}

func TestService_Execute(t *testing.T) {
	checks := []Check{
		{ID: "test1", Title: "Test 1", Critical: true, Retries: 0},
		{ID: "test2", Title: "Test 2", Critical: false, Retries: 1},
	}

	t.Run("successful check", func(t *testing.T) {
		checker := &mockChecker{passed: true, message: "ok"}
		svc := NewService(checks, checker)

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
		svc := NewService(checks, checker)

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
		svc := NewService(checks, checker)

		svc.Execute(context.Background(), "test2")

		result, _ := svc.Result("test2")
		if result.Status != StatusDegraded {
			t.Errorf("Status = %v, want %v", result.Status, StatusDegraded)
		}
	})

	t.Run("recovery resets failures", func(t *testing.T) {
		checker := &mockChecker{passed: false, message: "error"}
		svc := NewService(checks, checker)

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
		{ID: "critical", Title: "Critical", Critical: true},
		{ID: "non-critical", Title: "Non-Critical", Critical: false},
	}

	t.Run("all pass", func(t *testing.T) {
		checker := &mockChecker{passed: true}
		svc := NewService(checks, checker)

		svc.Execute(context.Background(), "critical")
		svc.Execute(context.Background(), "non-critical")

		if status := svc.AggregateStatus(); status != StatusPass {
			t.Errorf("AggregateStatus() = %v, want %v", status, StatusPass)
		}
	})

	t.Run("critical fails", func(t *testing.T) {
		svc := NewService(checks, &mockChecker{passed: true})
		svc.Execute(context.Background(), "non-critical")

		svc2 := svc
		svc2.checker = &mockChecker{passed: false}
		svc2.Execute(context.Background(), "critical")

		if status := svc.AggregateStatus(); status != StatusFail {
			t.Errorf("AggregateStatus() = %v, want %v", status, StatusFail)
		}
	})

	t.Run("non-critical fails only", func(t *testing.T) {
		passChecker := &mockChecker{passed: true}
		failChecker := &mockChecker{passed: false}

		svc := NewService(checks, passChecker)
		svc.Execute(context.Background(), "critical")

		svc.checker = failChecker
		svc.Execute(context.Background(), "non-critical")

		if status := svc.AggregateStatus(); status != StatusDegraded {
			t.Errorf("AggregateStatus() = %v, want %v", status, StatusDegraded)
		}
	})
}

func TestService_Results(t *testing.T) {
	checks := []Check{
		{ID: "a", Title: "A"},
		{ID: "b", Title: "B"},
	}

	svc := NewService(checks, &mockChecker{passed: true})
	results := svc.Results()

	if len(results) != 2 {
		t.Errorf("Results() len = %v, want 2", len(results))
	}
}

func TestService_Checks(t *testing.T) {
	checks := []Check{
		{ID: "a", Title: "A"},
		{ID: "b", Title: "B"},
	}

	svc := NewService(checks, &mockChecker{})
	got := svc.Checks()

	if len(got) != 2 {
		t.Errorf("Checks() len = %v, want 2", len(got))
	}
}
