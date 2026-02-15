package health

import (
	"testing"
	"time"
)

func TestResult_Age(t *testing.T) {
	t.Run("zero time", func(t *testing.T) {
		r := Result{}
		if got := r.Age(); got != 0 {
			t.Errorf("Age() = %v, want 0", got)
		}
	})

	t.Run("non-zero time", func(t *testing.T) {
		r := Result{LastCheck: time.Now().Add(-5 * time.Second)}
		age := r.Age()
		if age < 5*time.Second || age > 6*time.Second {
			t.Errorf("Age() = %v, want ~5s", age)
		}
	})
}

func TestResult_ComputeStatus(t *testing.T) {
	tests := []struct {
		name                string
		critical            bool
		retries             int
		consecutiveFailures int
		passed              bool
		want                Status
	}{
		{"passed", true, 0, 0, true, StatusPass},
		{"critical fail no retries", true, 0, 0, false, StatusFail},
		{"critical within retries", true, 2, 0, false, StatusDegraded},
		{"critical within retries 2", true, 2, 1, false, StatusDegraded},
		{"critical exceeded retries", true, 2, 2, false, StatusFail},
		{"non-critical fail", false, 0, 0, false, StatusDegraded},
		{"non-critical exceeded retries", false, 2, 5, false, StatusDegraded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Result{
				Critical:            tt.critical,
				Retries:             tt.retries,
				ConsecutiveFailures: tt.consecutiveFailures,
			}
			if got := r.ComputeStatus(tt.passed); got != tt.want {
				t.Errorf("ComputeStatus(%v) = %v, want %v", tt.passed, got, tt.want)
			}
		})
	}
}
