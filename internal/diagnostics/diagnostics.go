// Package diagnostics inspects the environment Echo needs and reports what it
// found in terms a user can act on.
//
// Every probe is injected, so each outcome — present, missing, misconfigured —
// is testable on a machine that has none of the real tools installed.
package diagnostics

import (
	"fmt"
	"io"
	"strings"
)

// Necessity says whether Echo can work without a dependency.
type Necessity int

const (
	// Required means core operation is impossible without it.
	Required Necessity = iota
	// Optional means Echo still works, with reduced capability.
	Optional
	// Informational is context rather than a dependency, like the platform.
	Informational
)

func (n Necessity) String() string {
	switch n {
	case Required:
		return "required"
	case Optional:
		return "optional"
	default:
		return "info"
	}
}

// State is the outcome of a single check.
type State int

const (
	// StateOK means the dependency is present and usable.
	StateOK State = iota
	// StateMissing means it could not be found at all.
	StateMissing
	// StateMisconfigured means it was found but cannot be used as-is.
	StateMisconfigured
	// StateUnknown means the check could not reach a verdict.
	StateUnknown
)

func (s State) String() string {
	switch s {
	case StateOK:
		return "ok"
	case StateMissing:
		return "missing"
	case StateMisconfigured:
		return "misconfigured"
	default:
		return "unknown"
	}
}

// marker is the leading glyph for a state. ASCII only: the Windows console's
// default code page mangles anything else.
func (s State) marker() string {
	switch s {
	case StateOK:
		return "+"
	case StateMissing:
		return "-"
	case StateMisconfigured:
		return "!"
	default:
		return "?"
	}
}

// Check is one inspected thing.
type Check struct {
	// Name is what was inspected, such as "ffmpeg".
	Name string
	// Necessity says whether Echo needs it.
	Necessity Necessity
	// State is the verdict.
	State State
	// Detail is the evidence: a version, a path, or why it failed.
	Detail string
	// Remediation tells the user what to do. Required only when State is not OK.
	Remediation string
}

// blocking reports whether this check prevents Echo from working.
func (c Check) blocking() bool {
	return c.Necessity == Required && c.State != StateOK
}

// Report is the full set of checks, in the order they were run.
type Report struct {
	Checks []Check
}

// Blocking lists the required checks that failed.
func (r Report) Blocking() []Check {
	var blocking []Check
	for _, check := range r.Checks {
		if check.blocking() {
			blocking = append(blocking, check)
		}
	}

	return blocking
}

// Degraded lists the optional checks that failed.
func (r Report) Degraded() []Check {
	var degraded []Check
	for _, check := range r.Checks {
		if check.Necessity == Optional && check.State != StateOK {
			degraded = append(degraded, check)
		}
	}

	return degraded
}

// Render renders the report for a terminal, grouped by necessity so a user can
// tell at a glance what blocks them from what merely limits them.
func (r Report) Render(w io.Writer) {
	groups := []struct {
		necessity Necessity
		heading   string
	}{
		{Informational, "Environment"},
		{Required, "Required"},
		{Optional, "Optional"},
	}

	for _, group := range groups {
		checks := r.byNecessity(group.necessity)
		if len(checks) == 0 {
			continue
		}

		fmt.Fprintf(w, "%s\n", group.heading)
		width := nameWidth(checks)

		for _, check := range checks {
			fmt.Fprintf(w, "  %s %-*s  %s\n", check.State.marker(), width, check.Name, check.Detail)
			if check.State != StateOK && check.Remediation != "" {
				fmt.Fprintf(w, "    %*s  -> %s\n", width, "", check.Remediation)
			}
		}

		fmt.Fprintln(w)
	}

	r.writeSummary(w)
}

// writeSummary states the overall verdict plainly.
func (r Report) writeSummary(w io.Writer) {
	blocking, degraded := r.Blocking(), r.Degraded()

	switch {
	case len(blocking) > 0:
		fmt.Fprintf(w, "Not ready: %s.\n", joinNames(blocking))
		fmt.Fprintln(w, "Resolve the required items above, then run doctor again.")
	case len(degraded) > 0:
		fmt.Fprintf(w, "Ready, with limits: %s.\n", joinNames(degraded))
	default:
		fmt.Fprintln(w, "Ready.")
	}
}

func (r Report) byNecessity(necessity Necessity) []Check {
	var checks []Check
	for _, check := range r.Checks {
		if check.Necessity == necessity {
			checks = append(checks, check)
		}
	}

	return checks
}

func nameWidth(checks []Check) int {
	width := 0
	for _, check := range checks {
		if len(check.Name) > width {
			width = len(check.Name)
		}
	}

	return width
}

func joinNames(checks []Check) string {
	names := make([]string, 0, len(checks))
	for _, check := range checks {
		names = append(names, check.Name)
	}

	return strings.Join(names, ", ")
}
