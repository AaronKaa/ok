package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPChecker_Execute(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		check      Check
		wantPassed bool
		wantMsg    string
	}{
		{
			name:       "status match",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) },
			check:      Check{Method: "GET", ExpectStatus: 200},
			wantPassed: true,
			wantMsg:    "status 200",
		},
		{
			name:       "status mismatch",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) },
			check:      Check{Method: "GET", ExpectStatus: 200},
			wantPassed: false,
			wantMsg:    "expected status 200, got 500",
		},
		{
			name: "json path exists",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"ok"}`))
			},
			check:      Check{Method: "GET", ExpectStatus: 200, JSONPath: "status"},
			wantPassed: true,
			wantMsg:    `json_path "status" = ok`,
		},
		{
			name: "json path not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"other":"value"}`))
			},
			check:      Check{Method: "GET", ExpectStatus: 200, JSONPath: "status"},
			wantPassed: false,
			wantMsg:    `json_path "status" not found`,
		},
		{
			name: "json equals match",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"healthy"}`))
			},
			check:      Check{Method: "GET", ExpectStatus: 200, JSONPath: "status", JSONEquals: "healthy"},
			wantPassed: true,
			wantMsg:    `json_path "status" = "healthy"`,
		},
		{
			name: "json equals mismatch",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"unhealthy"}`))
			},
			check:      Check{Method: "GET", ExpectStatus: 200, JSONPath: "status", JSONEquals: "healthy"},
			wantPassed: false,
			wantMsg:    `json_path "status": expected "healthy", got "unhealthy"`,
		},
		{
			name: "POST method",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					w.WriteHeader(405)
					return
				}
				w.WriteHeader(200)
			},
			check:      Check{Method: "POST", ExpectStatus: 200},
			wantPassed: true,
		},
		{
			name: "nested json path",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"data":{"status":"ok"}}`))
			},
			check:      Check{Method: "GET", ExpectStatus: 200, JSONPath: "data.status", JSONEquals: "ok"},
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			checker := NewHTTPChecker(5 * time.Second)
			tt.check.URL = server.URL

			passed, msg, err := checker.Execute(context.Background(), tt.check)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if passed != tt.wantPassed {
				t.Errorf("Execute() passed = %v, want %v", passed, tt.wantPassed)
			}
			if tt.wantMsg != "" && msg != tt.wantMsg {
				t.Errorf("Execute() msg = %q, want %q", msg, tt.wantMsg)
			}
		})
	}
}

func TestHTTPChecker_Execute_ConnectionError(t *testing.T) {
	checker := NewHTTPChecker(1 * time.Second)
	check := Check{
		URL:          "http://localhost:59999", // unlikely to be listening
		Method:       "GET",
		ExpectStatus: 200,
	}

	passed, msg, err := checker.Execute(context.Background(), check)
	if err != nil {
		t.Fatalf("Execute() should not return error for connection issues, got %v", err)
	}
	if passed {
		t.Error("Execute() should return passed=false for connection error")
	}
	if msg == "" {
		t.Error("Execute() should return error message")
	}
}

func TestHTTPChecker_Execute_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	checker := NewHTTPChecker(100 * time.Millisecond)
	check := Check{URL: server.URL, Method: "GET", ExpectStatus: 200}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	passed, _, err := checker.Execute(ctx, check)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if passed {
		t.Error("Execute() should return passed=false for timeout")
	}
}
