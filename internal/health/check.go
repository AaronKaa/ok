package health

import (
	"time"

	"github.com/AaronKaa/ok/internal/config"
)

type Check struct {
	ID           string
	Title        string
	URL          string
	Method       string
	Interval     time.Duration
	ExpectStatus int
	JSONPath     string
	JSONEquals   string
	Critical     bool
	Retries      int
}

func NewCheck(def config.CheckDefinition) Check {
	return Check{
		ID:           def.ID,
		Title:        def.Title,
		URL:          def.URL,
		Method:       def.Method,
		Interval:     def.Interval,
		ExpectStatus: def.ExpectStatus,
		JSONPath:     def.JSONPath,
		JSONEquals:   def.JSONEquals,
		Critical:     def.IsCritical(),
		Retries:      def.Retries,
	}
}
