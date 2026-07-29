package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jwogrady/echo/internal/buildinfo"
	"github.com/jwogrady/echo/internal/config"
	"github.com/jwogrady/echo/internal/worker"
)

// probeTimeout bounds each external tool invocation. A hung nvidia-smi must not
// hang doctor, which is the command a stuck user reaches for first.
const probeTimeout = 5 * time.Second

// Tool is an external program Echo depends on.
type Tool struct {
	// Name is what the report calls it.
	Name string
	// Commands are the executables to try, in order. Most tools have one; the
	// Python interpreter is named differently per platform.
	Commands []string
	// Necessity says whether Echo can work without it.
	Necessity Necessity
	// VersionArgs produces version output, such as {"-version"}.
	VersionArgs []string
	// Remediation tells the user how to install it.
	Remediation string
}

// tools are the external programs doctor inspects, in report order.
//
// FFmpeg and ffprobe are required because the overview makes them an explicit
// prerequisite for the first release. uv and Python are required because the
// worker is a managed uv environment. NVIDIA tooling is optional: its absence
// means transcription cannot run, but Echo can still be installed, inspected,
// and used to import audio, and doctor must stay useful on a machine without a
// GPU rather than refusing to report.
func toolsFor(goos string) []Tool {
	// Windows ships the interpreter as python.exe; elsewhere python3 is the
	// canonical name and a bare "python" may be absent or a Python 2 relic.
	python := []string{"python3", "python"}
	if goos == "windows" {
		python = []string{"python", "python3"}
	}

	return []Tool{
		{
			Name:        "ffmpeg",
			Commands:    []string{"ffmpeg"},
			Necessity:   Required,
			VersionArgs: []string{"-version"},
			Remediation: "install FFmpeg and ensure ffmpeg is on PATH (winget install Gyan.FFmpeg)",
		},
		{
			Name:        "ffprobe",
			Commands:    []string{"ffprobe"},
			Necessity:   Required,
			VersionArgs: []string{"-version"},
			Remediation: "ffprobe ships with FFmpeg; ensure the FFmpeg bin directory is on PATH",
		},
		{
			Name:        "uv",
			Commands:    []string{"uv"},
			Necessity:   Required,
			VersionArgs: []string{"--version"},
			Remediation: "install uv, which manages the Python worker (winget install astral-sh.uv)",
		},
		{
			Name:        "python",
			Commands:    python,
			Necessity:   Required,
			VersionArgs: []string{"--version"},
			Remediation: "install Python 3.12 or newer, or let uv manage it (uv python install 3.12)",
		},
		{
			Name:        "nvidia-smi",
			Commands:    []string{"nvidia-smi"},
			Necessity:   Optional,
			VersionArgs: []string{"--query-gpu=name,driver_version", "--format=csv,noheader"},
			Remediation: "install the NVIDIA driver; without it GPU transcription cannot run",
		},
	}
}

// Environment is everything doctor reads from outside itself.
type Environment struct {
	// Build identifies the running binary.
	Build buildinfo.Info
	// GOOS and GOARCH name the platform.
	GOOS, GOARCH string
	// LookPath resolves a command on PATH.
	LookPath func(string) (string, error)
	// RunTool captures a tool's version output.
	RunTool func(name string, args ...string) (string, error)
	// ResolvePaths reports Echo's data directory.
	ResolvePaths func() (config.Paths, error)
	// CheckWritable reports whether a directory can be written to.
	CheckWritable func(dir string) error
	// LocateWorker finds the Python worker project.
	LocateWorker func() (worker.Location, error)
}

// HostEnvironment inspects the real machine.
func HostEnvironment() Environment {
	return Environment{
		Build:         buildinfo.Current(),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		LookPath:      exec.LookPath,
		RunTool:       runTool,
		ResolvePaths:  config.Resolve,
		CheckWritable: CheckWritable,
		LocateWorker:  func() (worker.Location, error) { return worker.Locate(worker.HostLookup()) },
	}
}

// Inspect runs every check and returns the report.
func Inspect(env Environment) Report {
	report := Report{}

	report.Checks = append(report.Checks,
		Check{
			Name:      "echo",
			Necessity: Informational,
			State:     StateOK,
			Detail:    env.Build.Version + " (" + buildinfo.Name + ")",
		},
		Check{
			Name:      "platform",
			Necessity: Informational,
			State:     StateOK,
			Detail:    env.GOOS + "/" + env.GOARCH,
		},
		inspectDataDir(env),
	)

	for _, tool := range toolsFor(env.GOOS) {
		report.Checks = append(report.Checks, inspectTool(env, tool))
	}

	report.Checks = append(report.Checks, inspectWorker(env))

	return report
}

// inspectDataDir resolves the data directory and confirms Echo can write there.
// A read-only data root is misconfigured rather than missing: the location is
// known, it just cannot be used.
func inspectDataDir(env Environment) Check {
	check := Check{Name: "data directory", Necessity: Required}

	paths, err := env.ResolvePaths()
	if err != nil {
		check.State = StateMisconfigured
		check.Detail = err.Error()
		check.Remediation = "set " + config.EnvDataDir + " to a writable directory"

		return check
	}

	check.Detail = paths.Root

	if err := env.CheckWritable(paths.Root); err != nil {
		check.State = StateMisconfigured
		check.Detail = paths.Root + " (not writable: " + err.Error() + ")"
		check.Remediation = "grant write access, or set " + config.EnvDataDir + " to a writable directory"

		return check
	}

	check.State = StateOK
	check.Detail = paths.Root + " (writable)"

	return check
}

// inspectTool resolves a tool on PATH and asks it for its version. Being on
// PATH but failing to run is misconfigured, not missing — a broken install and
// an absent one need different advice.
func inspectTool(env Environment, tool Tool) Check {
	check := Check{Name: tool.Name, Necessity: tool.Necessity, Remediation: tool.Remediation}

	command, path, found := resolve(env, tool.Commands)
	if !found {
		check.State = StateMissing
		check.Detail = "not found on PATH (tried " + strings.Join(tool.Commands, ", ") + ")"

		return check
	}

	output, err := env.RunTool(command, tool.VersionArgs...)
	if err != nil {
		check.State = StateMisconfigured
		check.Detail = fmt.Sprintf("%s failed to run: %v", path, err)
		check.Remediation = "reinstall " + tool.Name + "; it is on PATH but not working"

		return check
	}

	check.State = StateOK
	check.Detail = firstLine(output)
	if command != tool.Name {
		check.Detail += " (" + command + ")"
	}

	return check
}

// resolve returns the first command that exists on PATH.
func resolve(env Environment, commands []string) (command, path string, found bool) {
	for _, candidate := range commands {
		if resolved, err := env.LookPath(candidate); err == nil {
			return candidate, resolved, true
		}
	}

	return "", "", false
}

// inspectWorker locates the Python worker and reports the contract version this
// CLI speaks. The worker's own version is checked when transcription runs; this
// build cannot invoke it, and doctor says so rather than implying it verified.
func inspectWorker(env Environment) Check {
	check := Check{Name: "worker", Necessity: Required}

	location, err := env.LocateWorker()
	if err != nil {
		check.State = StateMissing
		check.Detail = err.Error()
		check.Remediation = "set " + worker.EnvDir + " to the worker project directory"

		return check
	}

	check.State = StateOK
	check.Detail = fmt.Sprintf("%s (found %s; CLI speaks contract v%d)",
		location.Dir, location.Source, worker.ContractVersion)

	return check
}

// CheckWritable reports whether dir can be written to, creating it if needed.
// Probing with a real file is the only reliable answer on Windows, where
// permission bits do not tell the whole story.
func CheckWritable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create: %w", err)
	}

	probe, err := os.CreateTemp(dir, ".ekko-write-probe-*")
	if err != nil {
		return fmt.Errorf("cannot create a file: %w", err)
	}

	name := probe.Name()
	closeErr := probe.Close()
	removeErr := os.Remove(name)

	return errors.Join(closeErr, removeErr)
}

// runTool captures a command's output, merging stderr because several tools
// report their version there.
func runTool(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// firstLine trims a tool's output down to something a report can show.
func firstLine(output string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(output), "\n")

	return strings.TrimSpace(line)
}
