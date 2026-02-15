package server

import (
	"embed"
	"html/template"
	"io"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

var healthTemplate *template.Template

func init() {
	funcs := template.FuncMap{
		"statusColor": func(status string) string {
			switch status {
			case "pass":
				return "#22c55e"
			case "degraded":
				return "#eab308"
			case "fail":
				return "#ef4444"
			default:
				return "#6b7280"
			}
		},
		"statusBgColor": func(status string) string {
			switch status {
			case "pass":
				return "#dcfce7"
			case "degraded":
				return "#fef9c3"
			case "fail":
				return "#fee2e2"
			default:
				return "#f3f4f6"
			}
		},
		"formatAge": func(age float64) string {
			d := time.Duration(age * float64(time.Second))
			if d < time.Minute {
				return d.Round(time.Second).String()
			}
			if d < time.Hour {
				return d.Round(time.Second).String()
			}
			return d.Round(time.Minute).String()
		},
		"formatDuration": func(ms float64) string {
			d := time.Duration(ms * float64(time.Millisecond))
			if d < time.Millisecond {
				return d.Round(time.Microsecond).String()
			}
			return d.Round(time.Millisecond).String()
		},
		"upper": func(s string) string {
			return template.HTMLEscapeString(s)
		},
	}

	healthTemplate = template.Must(template.New("health.html").Funcs(funcs).ParseFS(templateFS, "templates/health.html"))
}

type TemplateData struct {
	Response        HealthResponse
	RefreshInterval int
}

func RenderHealth(w io.Writer, data TemplateData) error {
	return healthTemplate.Execute(w, data)
}
