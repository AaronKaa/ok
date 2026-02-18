package config

import (
	"fmt"
	"net/url"
	"time"
)

func Validate(cfg *Config) error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535, got %d", cfg.Port)
	}

	if len(cfg.Checks) == 0 {
		return fmt.Errorf("no health checks configured; set CHECKS or CHECKS_* environment variables")
	}

	for i := range cfg.Checks {
		if err := validateAndApplyDefaults(&cfg.Checks[i]); err != nil {
			return fmt.Errorf("check %q: %w", cfg.Checks[i].ID, err)
		}
	}

	return nil
}

func validateAndApplyDefaults(check *CheckDefinition) error {
	if check.Title == "" {
		return fmt.Errorf("title is required")
	}

	if check.Type == "" {
		check.Type = CheckTypeHTTP
	}

	switch check.Type {
	case CheckTypeHTTP:
		if err := validateHTTPCheck(check); err != nil {
			return err
		}
	case CheckTypeDocker:
		if err := validateDockerCheck(check); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid check type %q, must be %q or %q", check.Type, CheckTypeHTTP, CheckTypeDocker)
	}

	if err := validateCommonFields(check); err != nil {
		return err
	}

	return nil
}

func validateHTTPCheck(check *CheckDefinition) error {
	if check.URL == "" {
		return fmt.Errorf("url is required for http checks")
	}

	u, err := url.Parse(check.URL)
	if err != nil {
		return fmt.Errorf("invalid url %q: %w", check.URL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https, got %q", u.Scheme)
	}

	if check.Method == "" {
		check.Method = "GET"
	}

	if check.ExpectStatus == 0 {
		check.ExpectStatus = 200
	}
	if check.ExpectStatus < 100 || check.ExpectStatus > 599 {
		return fmt.Errorf("expect_status must be between 100 and 599, got %d", check.ExpectStatus)
	}

	if check.JSONEquals != "" && check.JSONPath == "" {
		return fmt.Errorf("json_equals requires json_path to be set")
	}

	return nil
}

func validateDockerCheck(check *CheckDefinition) error {
	if check.Container == "" {
		return fmt.Errorf("container is required for docker checks")
	}
	return nil
}

func validateCommonFields(check *CheckDefinition) error {
	if check.IntervalStr == "" {
		check.IntervalStr = "30s"
	}
	d, err := time.ParseDuration(check.IntervalStr)
	if err != nil {
		return fmt.Errorf("invalid interval %q: %w", check.IntervalStr, err)
	}
	if d < time.Second {
		return fmt.Errorf("interval must be at least 1s, got %s", check.IntervalStr)
	}
	check.Interval = d

	if check.Retries < 0 {
		return fmt.Errorf("retries must be >= 0, got %d", check.Retries)
	}

	return nil
}
