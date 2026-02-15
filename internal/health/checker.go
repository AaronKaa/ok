package health

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tidwall/gjson"
)

type Checker interface {
	Execute(ctx context.Context, check Check) (passed bool, message string, err error)
}

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker(timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *HTTPChecker) Execute(ctx context.Context, check Check) (passed bool, message string, err error) {
	req, err := http.NewRequestWithContext(ctx, check.Method, check.URL, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != check.ExpectStatus {
		return false, fmt.Sprintf("expected status %d, got %d", check.ExpectStatus, resp.StatusCode), nil
	}

	if check.JSONPath == "" {
		return true, fmt.Sprintf("status %d", resp.StatusCode), nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Sprintf("failed to read response body: %v", err), nil
	}

	result := gjson.GetBytes(body, check.JSONPath)
	if !result.Exists() {
		return false, fmt.Sprintf("json_path %q not found", check.JSONPath), nil
	}

	if check.JSONEquals != "" {
		actual := result.String()
		if actual != check.JSONEquals {
			return false, fmt.Sprintf("json_path %q: expected %q, got %q", check.JSONPath, check.JSONEquals, actual), nil
		}
		return true, fmt.Sprintf("json_path %q = %q", check.JSONPath, actual), nil
	}

	return true, fmt.Sprintf("json_path %q = %s", check.JSONPath, result.String()), nil
}
