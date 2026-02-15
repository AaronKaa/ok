package webhook

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/AaronKaa/ok/internal/config"
	"github.com/AaronKaa/ok/internal/health"
)

type Payload struct {
	Status         string        `json:"status"`
	PreviousStatus string        `json:"previous_status"`
	Timestamp      time.Time     `json:"timestamp"`
	FailedChecks   []CheckStatus `json:"failed_checks"`
}

type CheckStatus struct {
	ID                  string  `json:"id"`
	Title               string  `json:"title"`
	Status              string  `json:"status"`
	Critical            bool    `json:"critical"`
	Message             string  `json:"message"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	DurationMs          float64 `json:"duration_ms"`
}

type Client struct {
	cfg        config.WebhookConfig
	httpClient *http.Client
}

func New(cfg config.WebhookConfig) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Enabled() bool {
	return c.cfg.URL != ""
}

func (c *Client) Continuous() bool {
	return c.cfg.Continuous
}

func (c *Client) Notify(status, previousStatus health.Status, failedChecks []health.Result) {
	payload := Payload{
		Status:         string(status),
		PreviousStatus: string(previousStatus),
		Timestamp:      time.Now(),
		FailedChecks:   make([]CheckStatus, 0, len(failedChecks)),
	}

	for _, check := range failedChecks {
		payload.FailedChecks = append(payload.FailedChecks, CheckStatus{
			ID:                  check.CheckID,
			Title:               check.Title,
			Status:              string(check.Status),
			Critical:            check.Critical,
			Message:             check.Message,
			ConsecutiveFailures: check.ConsecutiveFailures,
			DurationMs:          float64(check.Duration.Microseconds()) / 1000,
		})
	}

	if err := c.Send(payload); err != nil {
		slog.Error("failed to send webhook", "error", err)
	}
}

func (c *Client) Send(payload Payload) error {
	if !c.Enabled() {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	slog.Info("webhook sent",
		"url", c.cfg.URL,
		"status", payload.Status,
		"failed_checks", len(payload.FailedChecks),
	)

	return nil
}

func (c *Client) applyAuth(req *http.Request) {
	switch c.cfg.AuthType {
	case "basic":
		auth := base64.StdEncoding.EncodeToString(
			[]byte(c.cfg.AuthUser + ":" + c.cfg.AuthPass),
		)
		req.Header.Set("Authorization", "Basic "+auth)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+c.cfg.AuthToken)
	case "header":
		if c.cfg.AuthHeaderName != "" {
			req.Header.Set(c.cfg.AuthHeaderName, c.cfg.AuthHeaderValue)
		}
	}
}
