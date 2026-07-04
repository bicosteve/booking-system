package health

import (
	"context"
	"time"
)

// Checker performs a single dependency reachability probe.
type Checker struct {
	// Name identifies the dependency in the report (e.g. "mysql").
	Name string
	// Disabled marks the dependency as intentionally off. Disabled checkers
	// are reported as "disabled" and never affect overall health.
	Disabled bool
	// Ping is invoked with a context carrying a per-checker timeout. A nil
	// Ping is treated as an unreachable dependency.
	Ping func(context.Context) error
}

// Result is the outcome of a single checker.
type Result struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "up" | "down" | "disabled"
	Error  string `json:"error,omitempty"`
}

// Report aggregates checker results.
type Report struct {
	Status string   `json:"status"` // "healthy" | "unhealthy"
	Checks []Result `json:"checks"`
}

// Check runs each checker's Ping (bounded by a per-checker 2s timeout derived
// from ctx) and returns a Report. Overall Status is "healthy" iff every
// enabled checker is "up". Disabled checkers are not executed.
func Check(ctx context.Context, checkers []Checker) Report {
	report := Report{Checks: make([]Result, 0, len(checkers))}
	healthy := true

	for _, c := range checkers {
		if c.Disabled {
			report.Checks = append(report.Checks, Result{Name: c.Name, Status: "disabled"})
			continue
		}
		status := "up"
		errMsg := ""
		if c.Ping == nil {
			status = "down"
			errMsg = "ping function not configured"
			healthy = false
		} else {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := c.Ping(pingCtx)
			cancel()
			if err != nil {
				status = "down"
				errMsg = err.Error()
				healthy = false
			}
		}
		report.Checks = append(report.Checks, Result{Name: c.Name, Status: status, Error: errMsg})
	}

	if healthy {
		report.Status = "healthy"
	} else {
		report.Status = "unhealthy"
	}
	return report
}
