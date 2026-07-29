package diagnostics

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwogrady/echo/internal/buildinfo"
	"github.com/jwogrady/echo/internal/config"
	"github.com/jwogrady/echo/internal/worker"
)

// healthyEnvironment reports every dependency as present. Individual tests break
// one thing at a time so a failure names exactly what changed.
func healthyEnvironment() Environment {
	return Environment{
		Build:    buildinfo.Info{Version: "1.2.3"},
		GOOS:     "linux",
		GOARCH:   "amd64",
		LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		RunTool: func(name string, _ ...string) (string, error) {
			return name + " version 1.0\nsecond line ignored", nil
		},
		ResolvePaths:  func() (config.Paths, error) { return config.Paths{Root: "/data/echo"}, nil },
		CheckWritable: func(string) error { return nil },
		LocateWorker: func() (worker.Location, error) {
			return worker.Location{Dir: "/opt/echo/worker", Source: "beside the executable"}, nil
		},
	}
}

// find returns the named check, failing the test when it is absent.
func find(t *testing.T, report Report, name string) Check {
	t.Helper()

	for _, check := range report.Checks {
		if check.Name == name {
			return check
		}
	}

	t.Fatalf("no check named %q in %v", name, report.Checks)

	return Check{}
}

func TestHealthyEnvironmentIsReady(t *testing.T) {
	report := Inspect(healthyEnvironment())

	if blocking := report.Blocking(); len(blocking) != 0 {
		t.Errorf("Blocking() = %v, want none", blocking)
	}
	if degraded := report.Degraded(); len(degraded) != 0 {
		t.Errorf("Degraded() = %v, want none", degraded)
	}

	var out bytes.Buffer
	report.Render(&out)
	if !strings.Contains(out.String(), "Ready.") {
		t.Errorf("report = %q, want it to say Ready", out.String())
	}
}

// The criterion is that required and optional dependencies are reported
// separately.
func TestRequiredAndOptionalAreReportedSeparately(t *testing.T) {
	report := Inspect(healthyEnvironment())

	var out bytes.Buffer
	report.Render(&out)
	rendered := out.String()

	required := strings.Index(rendered, "Required")
	optional := strings.Index(rendered, "Optional")

	if required < 0 || optional < 0 {
		t.Fatalf("report = %q, want both a Required and an Optional section", rendered)
	}
	if required > optional {
		t.Error("Required must be reported before Optional")
	}

	if got := find(t, report, "ffmpeg").Necessity; got != Required {
		t.Errorf("ffmpeg necessity = %v, want required", got)
	}
	if got := find(t, report, "nvidia-smi").Necessity; got != Optional {
		t.Errorf("nvidia-smi necessity = %v, want optional", got)
	}
}

func TestEveryRequiredDependencyIsChecked(t *testing.T) {
	report := Inspect(healthyEnvironment())

	// The acceptance criterion names each of these explicitly.
	for _, name := range []string{"ffmpeg", "ffprobe", "uv", "python", "nvidia-smi", "worker", "data directory"} {
		find(t, report, name)
	}
}

func TestMissingToolIsReportedWithRemediation(t *testing.T) {
	env := healthyEnvironment()
	env.LookPath = func(name string) (string, error) {
		if name == "ffmpeg" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + name, nil
	}

	report := Inspect(env)
	check := find(t, report, "ffmpeg")

	if check.State != StateMissing {
		t.Errorf("State = %v, want missing", check.State)
	}
	if check.Remediation == "" {
		t.Error("a failing check must carry remediation")
	}
	if len(report.Blocking()) != 1 {
		t.Errorf("Blocking() = %v, want exactly ffmpeg", report.Blocking())
	}
}

// A tool on PATH that will not run needs different advice than an absent one.
func TestBrokenToolIsMisconfiguredNotMissing(t *testing.T) {
	env := healthyEnvironment()
	env.RunTool = func(name string, _ ...string) (string, error) {
		if name == "uv" {
			return "", errors.New("exec format error")
		}
		return name + " 1.0", nil
	}

	check := find(t, Inspect(env), "uv")

	if check.State != StateMisconfigured {
		t.Errorf("State = %v, want misconfigured", check.State)
	}
	if !strings.Contains(check.Remediation, "reinstall") {
		t.Errorf("Remediation = %q, want it to advise reinstalling", check.Remediation)
	}
}

// Missing GPU tooling limits Echo but must not make doctor refuse to report.
func TestMissingOptionalToolDoesNotBlock(t *testing.T) {
	env := healthyEnvironment()
	env.LookPath = func(name string) (string, error) {
		if name == "nvidia-smi" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + name, nil
	}

	report := Inspect(env)

	if len(report.Blocking()) != 0 {
		t.Errorf("Blocking() = %v, want none", report.Blocking())
	}
	if len(report.Degraded()) != 1 {
		t.Errorf("Degraded() = %v, want exactly nvidia-smi", report.Degraded())
	}

	var out bytes.Buffer
	report.Render(&out)
	if !strings.Contains(out.String(), "Ready, with limits") {
		t.Errorf("report = %q, want a degraded summary", out.String())
	}
}

func TestPythonIsProbedByPlatform(t *testing.T) {
	tests := []struct {
		goos      string
		available string
		want      string
	}{
		{goos: "linux", available: "python3", want: "python3"},
		{goos: "darwin", available: "python3", want: "python3"},
		{goos: "windows", available: "python", want: "python"},
	}

	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			env := healthyEnvironment()
			env.GOOS = test.goos

			var probed string
			env.LookPath = func(name string) (string, error) {
				if name != test.available {
					return "", errors.New("not found")
				}
				return "/usr/bin/" + name, nil
			}
			env.RunTool = func(name string, _ ...string) (string, error) {
				if name == test.available {
					probed = name
				}
				return "Python 3.12.0", nil
			}

			check := find(t, Inspect(env), "python")

			if check.State != StateOK {
				t.Fatalf("State = %v (%s), want ok", check.State, check.Detail)
			}
			if probed != test.want {
				t.Errorf("probed %q, want %q", probed, test.want)
			}
		})
	}
}

// Linux and macOS commonly have no bare "python"; reporting that as missing
// would be a false failure for most developers.
func TestPythonFoundAsPython3IsNotAFailure(t *testing.T) {
	env := healthyEnvironment()
	env.GOOS = "linux"
	env.LookPath = func(name string) (string, error) {
		if name == "python" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + name, nil
	}

	check := find(t, Inspect(env), "python")

	if check.State != StateOK {
		t.Errorf("State = %v, want ok when python3 exists", check.State)
	}
	if !strings.Contains(check.Detail, "python3") {
		t.Errorf("Detail = %q, want it to name the command actually used", check.Detail)
	}
}

func TestUnresolvableDataDirIsMisconfigured(t *testing.T) {
	env := healthyEnvironment()
	env.ResolvePaths = func() (config.Paths, error) { return config.Paths{}, config.ErrNoDataDir }

	check := find(t, Inspect(env), "data directory")

	if check.State != StateMisconfigured {
		t.Errorf("State = %v, want misconfigured", check.State)
	}
	if !strings.Contains(check.Remediation, config.EnvDataDir) {
		t.Errorf("Remediation = %q, want it to name %s", check.Remediation, config.EnvDataDir)
	}
}

// A known but unwritable location is misconfigured, not missing.
func TestUnwritableDataDirIsMisconfigured(t *testing.T) {
	env := healthyEnvironment()
	env.CheckWritable = func(string) error { return errors.New("permission denied") }

	check := find(t, Inspect(env), "data directory")

	if check.State != StateMisconfigured {
		t.Errorf("State = %v, want misconfigured", check.State)
	}
	if !strings.Contains(check.Detail, "not writable") {
		t.Errorf("Detail = %q, want it to say the directory is not writable", check.Detail)
	}
}

func TestMissingWorkerIsBlockingWithRemediation(t *testing.T) {
	env := healthyEnvironment()
	env.LocateWorker = func() (worker.Location, error) { return worker.Location{}, worker.ErrNotFound }

	report := Inspect(env)
	check := find(t, report, "worker")

	if check.State != StateMissing {
		t.Errorf("State = %v, want missing", check.State)
	}
	if !strings.Contains(check.Remediation, worker.EnvDir) {
		t.Errorf("Remediation = %q, want it to name %s", check.Remediation, worker.EnvDir)
	}
	if len(report.Blocking()) != 1 {
		t.Errorf("Blocking() = %v, want exactly the worker", report.Blocking())
	}
}

func TestWorkerCheckReportsTheContractVersion(t *testing.T) {
	check := find(t, Inspect(healthyEnvironment()), "worker")

	if !strings.Contains(check.Detail, "contract v1") {
		t.Errorf("Detail = %q, want the contract version the CLI speaks", check.Detail)
	}
}

// Every failing check must be actionable, whatever broke it.
func TestEveryFailingCheckCarriesRemediation(t *testing.T) {
	env := healthyEnvironment()
	env.LookPath = func(string) (string, error) { return "", errors.New("not found") }
	env.ResolvePaths = func() (config.Paths, error) { return config.Paths{}, config.ErrNoDataDir }
	env.LocateWorker = func() (worker.Location, error) { return worker.Location{}, worker.ErrNotFound }

	report := Inspect(env)

	for _, check := range report.Checks {
		if check.State == StateOK || check.Necessity == Informational {
			continue
		}
		if check.Remediation == "" {
			t.Errorf("%s failed with no remediation", check.Name)
		}
	}
}

func TestReportNamesWhatBlocksTheUser(t *testing.T) {
	env := healthyEnvironment()
	env.LookPath = func(name string) (string, error) {
		if name == "ffmpeg" || name == "ffprobe" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + name, nil
	}

	var out bytes.Buffer
	Inspect(env).Render(&out)
	rendered := out.String()

	if !strings.Contains(rendered, "Not ready: ffmpeg, ffprobe.") {
		t.Errorf("report = %q, want it to name both blockers", rendered)
	}
}

// The Windows console's default code page mangles non-ASCII, and doctor is the
// command a user runs when things are already going wrong.
func TestReportIsASCIIOnly(t *testing.T) {
	env := healthyEnvironment()
	env.LookPath = func(string) (string, error) { return "", errors.New("not found") }

	var out bytes.Buffer
	Inspect(env).Render(&out)

	for index, r := range out.String() {
		if r > 127 {
			t.Errorf("byte %d is non-ASCII (%q); Windows consoles mangle it", index, r)
		}
	}
}

func TestCheckWritableAcceptsAUsableDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "created", "on", "demand")

	if err := CheckWritable(dir); err != nil {
		t.Fatalf("CheckWritable() error = %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the probed directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("CheckWritable left %d file(s) behind", len(entries))
	}
}

func TestCheckWritableRejectsAFileMasqueradingAsADirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	if err := CheckWritable(path); err == nil {
		t.Error("expected an error for a path that is a file")
	}
}

func TestFirstLineTrimsToolChatter(t *testing.T) {
	if got := firstLine("  ffmpeg version 6.0\nbuilt with gcc\n"); got != "ffmpeg version 6.0" {
		t.Errorf("firstLine() = %q", got)
	}
}
