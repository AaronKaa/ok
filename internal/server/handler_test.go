package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AaronKaa/ok/internal/health"
)

type mockHealthService struct {
	results []health.Result
	status  health.Status
}

func (m *mockHealthService) Results() []health.Result {
	return m.results
}

func (m *mockHealthService) AggregateStatus() health.Status {
	return m.status
}

func TestHandler_ServeHTTP(t *testing.T) {
	svc := &mockHealthService{
		results: []health.Result{
			{CheckID: "test", Title: "Test", Status: health.StatusPass, Critical: true, LastCheck: time.Now()},
		},
		status: health.StatusPass,
	}

	handler := NewHandler(svc, 10*time.Second)

	t.Run("default returns HTML", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Errorf("Content-Type = %q, want text/html", ct)
		}
	})

	t.Run("Accept header JSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var resp HealthResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if resp.Status != "pass" {
			t.Errorf("status = %q, want pass", resp.Status)
		}
	})

	t.Run("format query param JSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health?format=json", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
	})

	t.Run("format query param HTML", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health?format=html", nil)
		req.Header.Set("Accept", "application/json") // Should be overridden
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Errorf("Content-Type = %q, want text/html", ct)
		}
	})

	t.Run("root path works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("404 for unknown path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unknown", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}

func TestHandler_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		status     health.Status
		wantStatus int
	}{
		{"pass", health.StatusPass, http.StatusOK},
		{"degraded", health.StatusDegraded, http.StatusOK},
		{"fail", health.StatusFail, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockHealthService{status: tt.status}
			handler := NewHandler(svc, 0)

			req := httptest.NewRequest("GET", "/health?format=json", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_JSONResponse(t *testing.T) {
	now := time.Now()
	svc := &mockHealthService{
		results: []health.Result{
			{
				CheckID:             "api",
				Title:               "API Health",
				Status:              health.StatusPass,
				Critical:            true,
				LastCheck:           now,
				Duration:            150 * time.Millisecond,
				Message:             "status 200",
				ConsecutiveFailures: 0,
			},
		},
		status: health.StatusPass,
	}

	handler := NewHandler(svc, 10*time.Second)
	req := httptest.NewRequest("GET", "/health?format=json", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.Checks) != 1 {
		t.Fatalf("checks len = %d, want 1", len(resp.Checks))
	}

	check := resp.Checks[0]
	if check.ID != "api" {
		t.Errorf("ID = %q, want api", check.ID)
	}
	if check.Title != "API Health" {
		t.Errorf("Title = %q, want API Health", check.Title)
	}
	if check.Status != "pass" {
		t.Errorf("Status = %q, want pass", check.Status)
	}
	if !check.Critical {
		t.Error("Critical = false, want true")
	}
	if check.DurationMs < 149 || check.DurationMs > 151 {
		t.Errorf("DurationMs = %f, want ~150", check.DurationMs)
	}
	if check.Message != "status 200" {
		t.Errorf("Message = %q, want 'status 200'", check.Message)
	}
}

func TestHandler_HTMLContainsRefresh(t *testing.T) {
	svc := &mockHealthService{status: health.StatusPass}

	t.Run("with refresh", func(t *testing.T) {
		handler := NewHandler(svc, 10*time.Second)
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		body := rec.Body.String()
		if !strings.Contains(body, "setInterval(refresh, REFRESH_INTERVAL)") {
			t.Error("HTML should contain JavaScript refresh interval")
		}
	})

	t.Run("without refresh", func(t *testing.T) {
		handler := NewHandler(svc, 0)
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		body := rec.Body.String()
		if strings.Contains(body, "REFRESH_INTERVAL") {
			t.Error("HTML should not contain refresh script when interval is 0")
		}
	})
}

func TestSortChecks(t *testing.T) {
	checks := []CheckResponse{
		{ID: "a", Title: "Alpha", Status: "pass", Critical: true},
		{ID: "b", Title: "Beta", Status: "fail", Critical: false},
		{ID: "c", Title: "Charlie", Status: "degraded", Critical: true},
		{ID: "d", Title: "Delta", Status: "fail", Critical: true},
		{ID: "e", Title: "Echo", Status: "pass", Critical: false},
	}

	sortChecks(checks)

	// Expected order:
	// 1. fail + critical (Delta)
	// 2. fail + non-critical (Beta)
	// 3. degraded + critical (Charlie)
	// 4. pass + critical (Alpha)
	// 5. pass + non-critical (Echo)

	expected := []string{"Delta", "Beta", "Charlie", "Alpha", "Echo"}
	for i, exp := range expected {
		if checks[i].Title != exp {
			t.Errorf("checks[%d].Title = %q, want %q", i, checks[i].Title, exp)
		}
	}
}
