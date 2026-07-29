// Package buildinfo reports which build of Echo is running.
//
// Values are injected at link time by the release build; a plain `go build` or
// `go run` leaves them empty, so they fall back to what the Go toolchain
// already stamped into the binary. Either way the caller gets a complete
// answer rather than an empty string.
package buildinfo

import (
	"runtime"
	"runtime/debug"
)

// Injected via -ldflags "-X github.com/jwogrady/echo/internal/buildinfo.version=…".
var (
	version = ""
	commit  = ""
	date    = ""
)

// Name is the command users type. It is deliberately not "echo": that name is
// a shell builtin in zsh and bash and an alias for Write-Output in PowerShell,
// so a program installed under it can never be reached. See
// docs/plan/decisions/ADR-0002-command-name.md.
//
// This is the only place the command name is written. Everything that prints or
// registers it reads it from here.
const Name = "ekko"

// Unknown is reported for a field that neither the linker nor the embedded
// build metadata could supply.
const Unknown = "unknown"

// Info describes a single build of Echo.
type Info struct {
	// Version is the release version, or "devel" for an unreleased build.
	Version string
	// Commit is the full VCS revision the build came from.
	Commit string
	// Date is the build timestamp.
	Date string
	// GoVersion is the toolchain that produced the binary.
	GoVersion string
	// Platform is the target as GOOS/GOARCH.
	Platform string
}

// Current reports the running build.
func Current() Info {
	info := Info{
		Version:   version,
		Commit:    commit,
		Date:      date,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}

	if build, ok := debug.ReadBuildInfo(); ok {
		info.fillFromBuildInfo(build)
	}

	if info.Version == "" {
		info.Version = "devel"
	}
	if info.Commit == "" {
		info.Commit = Unknown
	}
	if info.Date == "" {
		info.Date = Unknown
	}

	return info
}

// fillFromBuildInfo supplies any field the linker left empty from the metadata
// the toolchain embedded. Linker values win: a release build states its own
// version deliberately.
func (i *Info) fillFromBuildInfo(build *debug.BuildInfo) {
	if i.Version == "" && build.Main.Version != "" && build.Main.Version != "(devel)" {
		i.Version = build.Main.Version
	}

	for _, setting := range build.Settings {
		switch setting.Key {
		case "vcs.revision":
			if i.Commit == "" {
				i.Commit = setting.Value
			}
		case "vcs.time":
			if i.Date == "" {
				i.Date = setting.Value
			}
		}
	}
}

// String renders the one-line form used by `ekko version`.
func (i Info) String() string {
	return Name + " " + i.Version + " (" + i.Commit + ", built " + i.Date + ", " + i.GoVersion + " " + i.Platform + ")"
}
