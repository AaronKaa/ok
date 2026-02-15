package health

import "testing"

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusPass, "pass"},
		{StatusDegraded, "degraded"},
		{StatusFail, "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusPass, true},
		{StatusDegraded, true},
		{StatusFail, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsHealthy(); got != tt.want {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}
