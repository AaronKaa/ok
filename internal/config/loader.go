package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func Load() (*Config, error) {
	cfg := &Config{
		Port:            8080,
		RefreshInterval: 10 * time.Second,
	}

	if portStr := os.Getenv("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT value %q: %w", portStr, err)
		}
		cfg.Port = port
	}

	if refreshStr := os.Getenv("REFRESH_INTERVAL"); refreshStr != "" {
		if refreshStr == "0" {
			cfg.RefreshInterval = 0
		} else {
			d, err := time.ParseDuration(refreshStr)
			if err != nil {
				return nil, fmt.Errorf("invalid REFRESH_INTERVAL value %q: %w", refreshStr, err)
			}
			cfg.RefreshInterval = d
		}
	}

	checks, err := loadChecks()
	if err != nil {
		return nil, err
	}
	cfg.Checks = checks

	return cfg, nil
}

func loadChecks() ([]CheckDefinition, error) {
	var checks []CheckDefinition

	if checksStr := os.Getenv("CHECKS"); checksStr != "" {
		parsed, err := parseChecksEnv("CHECKS", checksStr)
		if err != nil {
			return nil, err
		}
		checks = append(checks, parsed...)
	}

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "CHECKS_") {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := parts[0]
		value := parts[1]

		parsed, err := parseChecksEnv(name, value)
		if err != nil {
			return nil, err
		}
		checks = append(checks, parsed...)
	}

	return checks, nil
}

func parseChecksEnv(envName, value string) ([]CheckDefinition, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	var checks []CheckDefinition

	if strings.HasPrefix(value, "[") {
		if err := json.Unmarshal([]byte(value), &checks); err != nil {
			return nil, fmt.Errorf("failed to parse %s as JSON array: %w", envName, err)
		}
	} else if strings.HasPrefix(value, "{") {
		var check CheckDefinition
		if err := json.Unmarshal([]byte(value), &check); err != nil {
			return nil, fmt.Errorf("failed to parse %s as JSON object: %w", envName, err)
		}
		checks = append(checks, check)
	} else {
		return nil, fmt.Errorf("%s must be a JSON array or object, got: %s", envName, value[:min(20, len(value))])
	}

	suffix := ""
	if strings.HasPrefix(envName, "CHECKS_") {
		suffix = strings.ToLower(strings.TrimPrefix(envName, "CHECKS_"))
	}

	for i := range checks {
		if checks[i].ID == "" {
			if suffix != "" {
				if len(checks) > 1 {
					checks[i].ID = fmt.Sprintf("%s_%d", suffix, i)
				} else {
					checks[i].ID = suffix
				}
			} else {
				checks[i].ID = fmt.Sprintf("check_%d", i)
			}
		}
	}

	return checks, nil
}
