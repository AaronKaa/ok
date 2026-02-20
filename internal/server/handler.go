package server

import (
	"cmp"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/AaronKaa/ok/internal/health"
)

type HealthService interface {
	Results() []health.Result
	AggregateStatus() health.Status
}

type Handler struct {
	service         HealthService
	refreshInterval time.Duration
}

func NewHandler(service HealthService, refreshInterval time.Duration) *Handler {
	return &Handler{
		service:         service,
		refreshInterval: refreshInterval,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/health" && r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	response := h.buildResponse()

	format := r.URL.Query().Get("format")
	if format == "" {
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "application/json") {
			format = "json"
		} else {
			format = "html"
		}
	}

	statusCode := http.StatusOK
	switch response.Status {
	case "degraded":
		statusCode = http.StatusOK
	case "fail":
		statusCode = http.StatusServiceUnavailable
	}

	if format == "json" {
		h.serveJSON(w, response, statusCode)
	} else {
		h.serveHTML(w, response, statusCode)
	}
}

func (h *Handler) buildResponse() HealthResponse {
	results := h.service.Results()
	status := h.service.AggregateStatus()

	checks := make([]CheckResponse, 0, len(results))
	for _, result := range results {
		checks = append(checks, CheckResponse{
			ID:                  result.CheckID,
			Title:               result.Title,
			Status:              string(result.Status),
			Critical:            result.Critical,
			LastCheck:           formatTime(result.LastCheck),
			AgeSeconds:          result.Age().Seconds(),
			DurationMs:          float64(result.Duration.Microseconds()) / 1000,
			Message:             result.Message,
			ConsecutiveFailures: result.ConsecutiveFailures,
		})
	}

	sortChecks(checks)

	return HealthResponse{
		Status:    string(status),
		Timestamp: time.Now(),
		Checks:    checks,
	}
}

// sortChecks sorts checks by severity (fail > degraded > pass), then critical first, then alphabetically.
func sortChecks(checks []CheckResponse) {
	slices.SortFunc(checks, func(a, b CheckResponse) int {
		// Sort by status severity (fail=0, degraded=1, pass=2)
		if c := cmp.Compare(statusPriority(a.Status), statusPriority(b.Status)); c != 0 {
			return c
		}
		// Critical checks first
		if a.Critical != b.Critical {
			if a.Critical {
				return -1
			}
			return 1
		}
		// Alphabetically by title
		return cmp.Compare(a.Title, b.Title)
	})
}

func statusPriority(status string) int {
	switch status {
	case "fail":
		return 0
	case "degraded":
		return 1
	case "pass":
		return 2
	default:
		return 3
	}
}

func (h *Handler) serveJSON(w http.ResponseWriter, response HealthResponse, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) serveHTML(w http.ResponseWriter, response HealthResponse, statusCode int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	data := TemplateData{
		Response:        response,
		RefreshInterval: int(h.refreshInterval.Seconds()),
	}

	if err := RenderHealth(w, data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
