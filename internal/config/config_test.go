package config

import (
	"os"
	"testing"
	"time"
)

func TestCheckDefinition_IsCritical(t *testing.T) {
	tests := []struct {
		name     string
		critical *bool
		want     bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", new(true), true},
		{"explicit false", new(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CheckDefinition{Critical: tt.critical}
			if got := c.IsCritical(); got != tt.want {
				t.Errorf("IsCritical() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: &Config{Port: 8080, RefreshInterval: 10 * time.Second},
		},
		{
			name: "custom port",
			env:  map[string]string{"PORT": "3000"},
			want: &Config{Port: 3000, RefreshInterval: 10 * time.Second},
		},
		{
			name:    "invalid port",
			env:     map[string]string{"PORT": "invalid"},
			wantErr: true,
		},
		{
			name: "refresh interval disabled",
			env:  map[string]string{"REFRESH_INTERVAL": "0"},
			want: &Config{Port: 8080, RefreshInterval: 0},
		},
		{
			name: "custom refresh interval",
			env:  map[string]string{"REFRESH_INTERVAL": "30s"},
			want: &Config{Port: 8080, RefreshInterval: 30 * time.Second},
		},
		{
			name:    "invalid refresh interval",
			env:     map[string]string{"REFRESH_INTERVAL": "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			got, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %v, want %v", got.Port, tt.want.Port)
			}
			if got.RefreshInterval != tt.want.RefreshInterval {
				t.Errorf("RefreshInterval = %v, want %v", got.RefreshInterval, tt.want.RefreshInterval)
			}
		})
	}
}

func TestLoadChecks(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "no checks",
			env:       map[string]string{},
			wantCount: 0,
		},
		{
			name:      "single check array",
			env:       map[string]string{"CHECKS": `[{"title":"test","url":"http://test"}]`},
			wantCount: 1,
		},
		{
			name:      "multiple checks array",
			env:       map[string]string{"CHECKS": `[{"title":"a","url":"http://a"},{"title":"b","url":"http://b"}]`},
			wantCount: 2,
		},
		{
			name:      "single check object",
			env:       map[string]string{"CHECKS_DB": `{"title":"db","url":"http://db"}`},
			wantCount: 1,
		},
		{
			name: "combined checks",
			env: map[string]string{
				"CHECKS":    `[{"title":"main","url":"http://main"}]`,
				"CHECKS_DB": `{"title":"db","url":"http://db"}`,
			},
			wantCount: 2,
		},
		{
			name:    "invalid json",
			env:     map[string]string{"CHECKS": `invalid`},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			got, err := loadChecks()
			if (err != nil) != tt.wantErr {
				t.Errorf("loadChecks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantCount {
				t.Errorf("loadChecks() count = %v, want %v", len(got), tt.wantCount)
			}
		})
	}
}

func clearEnv() {
	os.Unsetenv("PORT")
	os.Unsetenv("REFRESH_INTERVAL")
	os.Unsetenv("CHECKS")
	for _, env := range os.Environ() {
		if len(env) > 7 && env[:7] == "CHECKS_" {
			os.Unsetenv(env[:len(env)-len("="+os.Getenv(env[:7]))])
		}
	}
}

//go:fix inline
func ptr(b bool) *bool { return new(b) }
