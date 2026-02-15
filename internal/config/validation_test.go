package config

import (
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "no checks",
			cfg:     &Config{Port: 8080},
			wantErr: true,
		},
		{
			name:    "invalid port low",
			cfg:     &Config{Port: 0, Checks: []CheckDefinition{{Title: "t", URL: "http://a"}}},
			wantErr: true,
		},
		{
			name:    "invalid port high",
			cfg:     &Config{Port: 70000, Checks: []CheckDefinition{{Title: "t", URL: "http://a"}}},
			wantErr: true,
		},
		{
			name:    "valid config",
			cfg:     &Config{Port: 8080, Checks: []CheckDefinition{{Title: "test", URL: "http://test"}}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAndApplyDefaults(t *testing.T) {
	tests := []struct {
		name    string
		check   CheckDefinition
		wantErr bool
		verify  func(t *testing.T, c *CheckDefinition)
	}{
		{
			name:    "missing title",
			check:   CheckDefinition{URL: "http://test"},
			wantErr: true,
		},
		{
			name:    "missing url",
			check:   CheckDefinition{Title: "test"},
			wantErr: true,
		},
		{
			name:    "invalid url scheme",
			check:   CheckDefinition{Title: "test", URL: "ftp://test"},
			wantErr: true,
		},
		{
			name:    "invalid interval",
			check:   CheckDefinition{Title: "test", URL: "http://test", IntervalStr: "invalid"},
			wantErr: true,
		},
		{
			name:    "interval too short",
			check:   CheckDefinition{Title: "test", URL: "http://test", IntervalStr: "500ms"},
			wantErr: true,
		},
		{
			name:    "invalid status code",
			check:   CheckDefinition{Title: "test", URL: "http://test", ExpectStatus: 999},
			wantErr: true,
		},
		{
			name:    "json_equals without json_path",
			check:   CheckDefinition{Title: "test", URL: "http://test", JSONEquals: "value"},
			wantErr: true,
		},
		{
			name:    "negative retries",
			check:   CheckDefinition{Title: "test", URL: "http://test", Retries: -1},
			wantErr: true,
		},
		{
			name:  "defaults applied",
			check: CheckDefinition{Title: "test", URL: "http://test"},
			verify: func(t *testing.T, c *CheckDefinition) {
				if c.Method != "GET" {
					t.Errorf("Method = %v, want GET", c.Method)
				}
				if c.Interval != 30*time.Second {
					t.Errorf("Interval = %v, want 30s", c.Interval)
				}
				if c.ExpectStatus != 200 {
					t.Errorf("ExpectStatus = %v, want 200", c.ExpectStatus)
				}
			},
		},
		{
			name:  "custom values preserved",
			check: CheckDefinition{Title: "test", URL: "https://test", Method: "POST", IntervalStr: "1m", ExpectStatus: 201},
			verify: func(t *testing.T, c *CheckDefinition) {
				if c.Method != "POST" {
					t.Errorf("Method = %v, want POST", c.Method)
				}
				if c.Interval != time.Minute {
					t.Errorf("Interval = %v, want 1m", c.Interval)
				}
				if c.ExpectStatus != 201 {
					t.Errorf("ExpectStatus = %v, want 201", c.ExpectStatus)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAndApplyDefaults(&tt.check)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAndApplyDefaults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.verify != nil {
				tt.verify(t, &tt.check)
			}
		})
	}
}
