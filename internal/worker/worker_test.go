package worker

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validManifest declares the entrypoint Locate insists on.
const validManifest = `[project]
name = "echo-worker"

[project.scripts]
echo-worker = "echo_worker.cli:main"
`

// lookupFor builds a Lookup over an in-memory filesystem.
func lookupFor(env map[string]string, exe, wd string, files map[string]string) Lookup {
	return Lookup{
		Getenv: func(key string) string { return env[key] },
		Executable: func() (string, error) {
			if exe == "" {
				return "", errors.New("executable path unavailable")
			}
			return exe, nil
		},
		Getwd: func() (string, error) {
			if wd == "" {
				return "", errors.New("working directory unavailable")
			}
			return wd, nil
		},
		ReadFile: func(name string) ([]byte, error) {
			contents, ok := files[filepath.Clean(name)]
			if !ok {
				return nil, os.ErrNotExist
			}
			return []byte(contents), nil
		},
	}
}

func TestLocateHonorsTheOverrideFirst(t *testing.T) {
	location, err := Locate(lookupFor(
		map[string]string{EnvDir: "/custom/worker"},
		"/opt/echo/ekko", "/home/jane/echo",
		map[string]string{
			"/custom/worker/pyproject.toml":         validManifest,
			"/home/jane/echo/worker/pyproject.toml": validManifest,
		},
	))
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if location.Dir != "/custom/worker" {
		t.Errorf("Dir = %q, want the override", location.Dir)
	}
	if location.Source != EnvDir {
		t.Errorf("Source = %q, want %q", location.Source, EnvDir)
	}
}

// A wrong override must fail loudly rather than falling back, or the user would
// never learn their setting was ignored.
func TestLocateFailsWhenTheOverrideIsWrong(t *testing.T) {
	_, err := Locate(lookupFor(
		map[string]string{EnvDir: "/nowhere"},
		"/opt/echo/ekko", "/home/jane/echo",
		map[string]string{"/home/jane/echo/worker/pyproject.toml": validManifest},
	))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), EnvDir) {
		t.Errorf("error = %q, want it to name %s", err, EnvDir)
	}
}

func TestLocateSearchesInOrder(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		wantDir    string
		wantSource string
	}{
		{
			name:       "beside the executable",
			files:      map[string]string{"/opt/echo/worker/pyproject.toml": validManifest},
			wantDir:    "/opt/echo/worker",
			wantSource: "beside the executable",
		},
		{
			name:       "above the executable",
			files:      map[string]string{"/opt/worker/pyproject.toml": validManifest},
			wantDir:    "/opt/worker",
			wantSource: "above the executable",
		},
		{
			name:       "working directory",
			files:      map[string]string{"/home/jane/echo/worker/pyproject.toml": validManifest},
			wantDir:    "/home/jane/echo/worker",
			wantSource: "working directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			location, err := Locate(lookupFor(nil, "/opt/echo/ekko", "/home/jane/echo", test.files))
			if err != nil {
				t.Fatalf("Locate() error = %v", err)
			}
			if location.Dir != test.wantDir {
				t.Errorf("Dir = %q, want %q", location.Dir, test.wantDir)
			}
			if location.Source != test.wantSource {
				t.Errorf("Source = %q, want %q", location.Source, test.wantSource)
			}
		})
	}
}

func TestLocateReportsWhereItLooked(t *testing.T) {
	_, err := Locate(lookupFor(nil, "/opt/echo/ekko", "/home/jane/echo", nil))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want it to wrap ErrNotFound", err)
	}
	for _, expected := range []string{"/opt/echo/worker", "/opt/worker", "/home/jane/echo/worker", EnvDir} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error = %q, want it to mention %q", err, expected)
		}
	}
}

// A directory named "worker" with an unrelated manifest is not Echo's worker.
func TestLocateRejectsAManifestWithoutTheEntrypoint(t *testing.T) {
	_, err := Locate(lookupFor(nil, "/opt/echo/ekko", "/home/jane/echo", map[string]string{
		"/opt/echo/worker/pyproject.toml": "[project]\nname = \"something-else\"\n",
	}))
	if err == nil {
		t.Fatal("expected an error: a manifest without the entrypoint is not the worker")
	}
}

func TestLocateSurvivesAnUnknownExecutablePath(t *testing.T) {
	location, err := Locate(lookupFor(nil, "", "/home/jane/echo", map[string]string{
		"/home/jane/echo/worker/pyproject.toml": validManifest,
	}))
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if location.Source != "working directory" {
		t.Errorf("Source = %q, want the working directory fallback", location.Source)
	}
}

// The committed worker/ must satisfy the same check Locate applies at runtime,
// so a change to either the manifest or the validation rule breaks a test rather
// than only breaking users. This one reads the real filesystem on purpose.
func TestTheCommittedWorkerIsValid(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolving the repository root: %v", err)
	}

	lookup := Lookup{
		Getenv:     func(string) string { return "" },
		Executable: func() (string, error) { return "", errors.New("not applicable") },
		Getwd:      func() (string, error) { return root, nil },
		ReadFile:   os.ReadFile,
	}

	location, err := Locate(lookup)
	if err != nil {
		t.Fatalf("the committed worker/ does not satisfy Locate: %v", err)
	}
	if want := filepath.Join(root, "worker"); location.Dir != want {
		t.Errorf("Dir = %q, want %q", location.Dir, want)
	}
}
