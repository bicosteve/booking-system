package health

import (
	"context"
	"fmt"
	"strings"
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

// Await calls Check every interval until all enabled checkers are up or
// timeout elapses. Returns nil once healthy; on timeout returns an error
// summarizing the failing checkers. Respects ctx cancellation.
func Await(ctx context.Context, checkers []Checker, interval, timeout time.Duration) error {
	if len(checkers) == 0 {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		last := Check(ctx, checkers)
		if last.Status == "healthy" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("dependencies not ready after %s: %s", timeout, summarize(last))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func summarize(r Report) string {
	var downs []string
	for _, c := range r.Checks {
		if c.Status == "down" {
			downs = append(downs, c.Name)
		}
	}
	if len(downs) == 0 {
		return "no details"
	}
	return "down: " + strings.Join(downs, ", ")
}
