package buildinfo

import (
	"runtime"
	"strings"
	"testing"
)

func TestCurrentReportsACompleteBuild(t *testing.T) {
	info := Current()

	if info.Version == "" {
		t.Error("Version is empty; an unstamped build should report devel")
	}
	if info.Commit == "" {
		t.Errorf("Commit is empty; want a revision or %q", Unknown)
	}
	if info.Date == "" {
		t.Errorf("Date is empty; want a timestamp or %q", Unknown)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
	if want := runtime.GOOS + "/" + runtime.GOARCH; info.Platform != want {
		t.Errorf("Platform = %q, want %q", info.Platform, want)
	}
}

// A `go test` binary carries no linker stamp, so this exercises the fallback.
func TestCurrentFallsBackToDevel(t *testing.T) {
	if version != "" {
		t.Skip("binary carries a linker-stamped version")
	}

	if got := Current().Version; got != "devel" {
		t.Errorf("Version = %q, want %q", got, "devel")
	}
}

func TestStringIncludesEveryField(t *testing.T) {
	info := Info{
		Version:   "0.1.0",
		Commit:    "abc1234",
		Date:      "2026-07-28T00:00:00Z",
		GoVersion: "go1.26.5",
		Platform:  "windows/amd64",
	}

	got := info.String()

	for _, want := range []string{"0.1.0", "abc1234", "2026-07-28T00:00:00Z", "go1.26.5", "windows/amd64"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, want it to contain %q", got, want)
		}
	}
	if !strings.HasPrefix(got, Name+" 0.1.0") {
		t.Errorf("String() = %q, want it to lead with the program and version", got)
	}
}

func TestFillFromBuildInfoDoesNotOverrideLinkerValues(t *testing.T) {
	info := Info{Version: "0.9.9", Commit: "stamped", Date: "stamped-date"}

	info.fillFromBuildInfo(buildInfoWith(map[string]string{
		"vcs.revision": "from-vcs",
		"vcs.time":     "from-vcs-time",
	}, "1.2.3"))

	if info.Version != "0.9.9" {
		t.Errorf("Version = %q, want the linker value %q", info.Version, "0.9.9")
	}
	if info.Commit != "stamped" {
		t.Errorf("Commit = %q, want the linker value %q", info.Commit, "stamped")
	}
	if info.Date != "stamped-date" {
		t.Errorf("Date = %q, want the linker value %q", info.Date, "stamped-date")
	}
}

func TestFillFromBuildInfoSuppliesMissingValues(t *testing.T) {
	var info Info

	info.fillFromBuildInfo(buildInfoWith(map[string]string{
		"vcs.revision": "from-vcs",
		"vcs.time":     "from-vcs-time",
	}, "1.2.3"))

	if info.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", info.Version, "1.2.3")
	}
	if info.Commit != "from-vcs" {
		t.Errorf("Commit = %q, want %q", info.Commit, "from-vcs")
	}
	if info.Date != "from-vcs-time" {
		t.Errorf("Date = %q, want %q", info.Date, "from-vcs-time")
	}
}

// A module built from a checkout with no tag reports "(devel)", which is not a
// version a user can act on.
func TestFillFromBuildInfoIgnoresDevelModuleVersion(t *testing.T) {
	var info Info

	info.fillFromBuildInfo(buildInfoWith(nil, "(devel)"))

	if info.Version != "" {
		t.Errorf("Version = %q, want it left empty so Current falls back", info.Version)
	}
}

// ADR-0002: the command name must never be "echo". It is a shell builtin in zsh
// and bash and an alias for Write-Output in PowerShell, so a program installed
// under that name is unreachable.
func TestNameIsNotAShellBuiltin(t *testing.T) {
	for _, reserved := range []string{"echo", "test", "printf", "cd", "set", "type"} {
		if Name == reserved {
			t.Fatalf("Name = %q, which shells resolve before PATH; see ADR-0002", Name)
		}
	}
}
