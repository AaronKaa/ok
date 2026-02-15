package server

import "time"

type HealthResponse struct {
	Status    string          `json:"status"`
	Timestamp time.Time       `json:"timestamp"`
	Checks    []CheckResponse `json:"checks"`
}

type CheckResponse struct {
	ID                  string  `json:"id"`
	Title               string  `json:"title"`
	Status              string  `json:"status"`
	Critical            bool    `json:"critical"`
	LastCheck           string  `json:"last_check"`
	AgeSeconds          float64 `json:"age_seconds"`
	DurationMs          float64 `json:"duration_ms"`
	Message             string  `json:"message"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
}
