// Package worker holds the Go side of the versioned subprocess contract with
// the Python GPU worker.
//
// ADR-0001 puts inference in Python and orchestration in Go, connected by a
// subprocess protocol. This package knows where the worker lives and which
// contract version this CLI speaks. It deliberately contains no Whisper
// dependency and no inference logic.
package worker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContractVersion is the protocol version this CLI speaks. The worker reports
// its own; the two must agree before transcription runs.
const ContractVersion = 1

// EnvDir overrides where Echo looks for the worker project.
const EnvDir = "ECHO_WORKER_DIR"

// dirName is the worker project directory inside a checkout.
const dirName = "worker"

// entrypoint is the console script the worker must declare.
const entrypoint = "echo-worker"

// ErrNotFound means no worker project could be located.
var ErrNotFound = errors.New("worker project not found")

// Location describes a worker project on disk.
type Location struct {
	// Dir is the project directory containing pyproject.toml.
	Dir string
	// Source explains how Dir was chosen, so a surprised user can tell why.
	Source string
}

// Lookup is the filesystem access Locate needs. Injecting it keeps discovery
// testable without building a checkout on disk.
type Lookup struct {
	// Getenv reads an environment variable.
	Getenv func(string) string
	// Executable reports the running binary's path.
	Executable func() (string, error)
	// Getwd reports the working directory.
	Getwd func() (string, error)
	// ReadFile reads a file's contents.
	ReadFile func(string) ([]byte, error)
}

// HostLookup reads the real process and filesystem.
func HostLookup() Lookup {
	return Lookup{
		Getenv:     os.Getenv,
		Executable: os.Executable,
		Getwd:      os.Getwd,
		ReadFile:   os.ReadFile,
	}
}

// Locate finds the worker project.
//
// The override is honored first. Otherwise Echo looks beside the running binary,
// then one directory above it, then in the working directory — covering a built
// executable sitting in a checkout and `go run` from the repository root.
func Locate(lookup Lookup) (Location, error) {
	if override := lookup.Getenv(EnvDir); override != "" {
		if err := validate(lookup, override); err != nil {
			return Location{}, fmt.Errorf("%s points at %s: %w", EnvDir, override, err)
		}

		return Location{Dir: override, Source: EnvDir}, nil
	}

	searched := candidates(lookup)

	tried := make([]string, 0, len(searched))
	for _, location := range searched {
		if validate(lookup, location.Dir) == nil {
			return location, nil
		}
		tried = append(tried, location.Dir)
	}

	return Location{}, fmt.Errorf("%w: looked in %s; set %s to point at it",
		ErrNotFound, strings.Join(tried, ", "), EnvDir)
}

// candidates lists the directories to search, in order.
func candidates(lookup Lookup) []Location {
	var found []Location

	if exe, err := lookup.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		found = append(found,
			Location{Dir: filepath.Join(exeDir, dirName), Source: "beside the executable"},
			Location{Dir: filepath.Clean(filepath.Join(exeDir, "..", dirName)), Source: "above the executable"},
		)
	}

	if wd, err := lookup.Getwd(); err == nil {
		found = append(found, Location{Dir: filepath.Join(wd, dirName), Source: "working directory"})
	}

	return found
}

// validate reports whether dir holds a worker project declaring the expected
// entrypoint. Checking the manifest rather than just the directory means a
// stray empty "worker" folder is not mistaken for the real thing.
func validate(lookup Lookup, dir string) error {
	manifest := filepath.Join(dir, "pyproject.toml")

	contents, err := lookup.ReadFile(manifest)
	if err != nil {
		return fmt.Errorf("no readable pyproject.toml: %w", err)
	}

	if !strings.Contains(string(contents), entrypoint) {
		return fmt.Errorf("pyproject.toml does not declare the %s entrypoint", entrypoint)
	}

	return nil
}
