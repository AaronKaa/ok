package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AaronKaa/ok/internal/config"
	"github.com/AaronKaa/ok/internal/health"
)

func TestClient_Enabled(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"enabled with url", "http://example.com", true},
		{"disabled without url", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(config.WebhookConfig{URL: tt.url})
			if got := c.Enabled(); got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Continuous(t *testing.T) {
	tests := []struct {
		name       string
		continuous bool
		want       bool
	}{
		{"continuous true", true, true},
		{"continuous false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(config.WebhookConfig{Continuous: tt.continuous})
			if got := c.Continuous(); got != tt.want {
				t.Errorf("Continuous() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Send(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		var received Payload
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Method = %v, want POST", r.Method)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", ct)
			}
			json.NewDecoder(r.Body).Decode(&received)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := New(config.WebhookConfig{URL: server.URL})
		payload := Payload{
			Status:         "fail",
			PreviousStatus: "pass",
			Timestamp:      time.Now(),
			FailedChecks: []CheckStatus{
				{ID: "test", Title: "Test", Status: "fail"},
			},
		}

		err := c.Send(payload)
		if err != nil {
			t.Errorf("Send() error = %v", err)
		}
		if received.Status != "fail" {
			t.Errorf("received status = %v, want fail", received.Status)
		}
		if len(received.FailedChecks) != 1 {
			t.Errorf("received checks = %v, want 1", len(received.FailedChecks))
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := New(config.WebhookConfig{URL: server.URL})
		err := c.Send(Payload{})

		if err == nil {
			t.Error("Send() expected error for 500 response")
		}
	})

	t.Run("disabled client", func(t *testing.T) {
		c := New(config.WebhookConfig{URL: ""})
		err := c.Send(Payload{})

		if err != nil {
			t.Errorf("Send() with disabled client should not error, got %v", err)
		}
	})
}

func TestClient_Auth(t *testing.T) {
	tests := []struct {
		name       string
		cfg        config.WebhookConfig
		checkAuth  func(t *testing.T, r *http.Request)
	}{
		{
			name: "basic auth",
			cfg: config.WebhookConfig{
				AuthType: "basic",
				AuthUser: "user",
				AuthPass: "pass",
			},
			checkAuth: func(t *testing.T, r *http.Request) {
				auth := r.Header.Get("Authorization")
				if auth == "" {
					t.Error("missing Authorization header")
				}
				if auth != "Basic dXNlcjpwYXNz" {
					t.Errorf("Authorization = %v, want Basic dXNlcjpwYXNz", auth)
				}
			},
		},
		{
			name: "bearer auth",
			cfg: config.WebhookConfig{
				AuthType:  "bearer",
				AuthToken: "mytoken",
			},
			checkAuth: func(t *testing.T, r *http.Request) {
				auth := r.Header.Get("Authorization")
				if auth != "Bearer mytoken" {
					t.Errorf("Authorization = %v, want Bearer mytoken", auth)
				}
			},
		},
		{
			name: "header auth",
			cfg: config.WebhookConfig{
				AuthType:        "header",
				AuthHeaderName:  "X-API-Key",
				AuthHeaderValue: "secret123",
			},
			checkAuth: func(t *testing.T, r *http.Request) {
				val := r.Header.Get("X-API-Key")
				if val != "secret123" {
					t.Errorf("X-API-Key = %v, want secret123", val)
				}
			},
		},
		{
			name: "no auth",
			cfg: config.WebhookConfig{
				AuthType: "none",
			},
			checkAuth: func(t *testing.T, r *http.Request) {
				auth := r.Header.Get("Authorization")
				if auth != "" {
					t.Errorf("Authorization should be empty, got %v", auth)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.checkAuth(t, r)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			tt.cfg.URL = server.URL
			c := New(tt.cfg)
			c.Send(Payload{})
		})
	}
}

func TestClient_Notify(t *testing.T) {
	var received Payload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(config.WebhookConfig{URL: server.URL})

	failedChecks := []health.Result{
		{
			CheckID:             "db",
			Title:               "Database",
			Status:              health.StatusFail,
			Critical:            true,
			Message:             "connection refused",
			ConsecutiveFailures: 3,
			Duration:            150 * time.Millisecond,
		},
	}

	c.Notify(health.StatusFail, health.StatusPass, failedChecks)

	if received.Status != "fail" {
		t.Errorf("Status = %v, want fail", received.Status)
	}
	if received.PreviousStatus != "pass" {
		t.Errorf("PreviousStatus = %v, want pass", received.PreviousStatus)
	}
	if len(received.FailedChecks) != 1 {
		t.Fatalf("FailedChecks len = %v, want 1", len(received.FailedChecks))
	}
	check := received.FailedChecks[0]
	if check.ID != "db" {
		t.Errorf("check ID = %v, want db", check.ID)
	}
	if check.Message != "connection refused" {
		t.Errorf("check Message = %v, want connection refused", check.Message)
	}
}
