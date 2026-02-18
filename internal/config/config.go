package config

import "time"

type Config struct {
	Port            int
	RefreshInterval time.Duration
	Checks          []CheckDefinition
	Webhook         WebhookConfig
}

type WebhookConfig struct {
	URL             string
	Continuous      bool   // If true, fire on every failure; if false, only on state changes
	AuthType        string // "none", "basic", "bearer", "header"
	AuthUser        string // For basic auth
	AuthPass        string // For basic auth
	AuthToken       string // For bearer auth
	AuthHeaderName  string // For custom header auth
	AuthHeaderValue string // For custom header auth
}

type CheckType string

const (
	CheckTypeHTTP   CheckType = "http"
	CheckTypeDocker CheckType = "docker"
)

type CheckDefinition struct {
	ID           string        `json:"-"`             // Generated identifier
	Type         CheckType     `json:"type"`          // Check type: "http" (default) or "docker"
	Title        string        `json:"title"`         // Display name
	URL          string        `json:"url"`           // Target URL (http checks)
	Method       string        `json:"method"`        // HTTP method (default: GET)
	Interval     time.Duration `json:"-"`             // Parsed interval
	IntervalStr  string        `json:"interval"`      // Raw interval string (default: 30s)
	ExpectStatus int           `json:"expect_status"` // Expected HTTP status (default: 200)
	JSONPath     string        `json:"json_path"`     // Optional JSON path to extract
	JSONEquals   string        `json:"json_equals"`   // Expected value at JSON path
	Critical     *bool         `json:"critical"`      // Whether failure causes aggregate fail (default: true)
	Retries      int           `json:"retries"`       // Retry count before marking failed (default: 0)
	Container    string        `json:"container"`     // Container name or ID (docker checks)
}

func (c CheckDefinition) IsCritical() bool {
	if c.Critical == nil {
		return true
	}
	return *c.Critical
}
