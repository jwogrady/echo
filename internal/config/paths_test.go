package config

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// envFrom builds an Environment backed by a map, so no test mutates process
// state and platforms other than the host can be exercised.
func envFrom(goos string, vars map[string]string, home string) Environment {
	return Environment{
		GOOS:   goos,
		Getenv: func(key string) string { return vars[key] },
		UserHomeDir: func() (string, error) {
			if home == "" {
				return "", errors.New("home directory unavailable")
			}
			return home, nil
		},
	}
}

// Assertions compose expected paths with filepath.Join, the same call the code
// uses. That verifies the composition — correct base location plus the right
// subdirectory — while leaving separator style to the standard library, which
// gets it right per-OS and cannot be exercised for Windows from a Linux host.
func TestResolveUsesPlatformConventions(t *testing.T) {
	tests := []struct {
		name string
		goos string
		vars map[string]string
		home string
		want string
	}{
		{
			name: "windows uses LOCALAPPDATA",
			goos: "windows",
			vars: map[string]string{"LOCALAPPDATA": `C:\Users\jane\AppData\Local`},
			home: `C:\Users\jane`,
			want: filepath.Join(`C:\Users\jane\AppData\Local`, "Echo"),
		},
		{
			name: "macos uses Application Support",
			goos: "darwin",
			home: "/Users/jane",
			want: filepath.Join("/Users/jane", "Library", "Application Support", "Echo"),
		},
		{
			name: "linux honors XDG_DATA_HOME",
			goos: "linux",
			vars: map[string]string{"XDG_DATA_HOME": "/home/jane/.local/share"},
			home: "/home/jane",
			want: filepath.Join("/home/jane/.local/share", "echo"),
		},
		{
			name: "linux falls back to the home directory",
			goos: "linux",
			home: "/home/jane",
			want: filepath.Join("/home/jane", ".local", "share", "echo"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paths, err := ResolveIn(envFrom(test.goos, test.vars, test.home))
			if err != nil {
				t.Fatalf("ResolveIn() error = %v", err)
			}
			if paths.Root != test.want {
				t.Errorf("Root = %q, want %q", paths.Root, test.want)
			}
		})
	}
}

// The override exists so automation and tests never touch real user data, so it
// must win on every platform.
func TestOverrideWinsOnEveryPlatform(t *testing.T) {
	for _, goos := range []string{"windows", "darwin", "linux"} {
		t.Run(goos, func(t *testing.T) {
			custom := t.TempDir()

			paths, err := ResolveIn(envFrom(goos, map[string]string{
				EnvDataDir:      custom,
				"LOCALAPPDATA":  `C:\Users\jane\AppData\Local`,
				"XDG_DATA_HOME": "/home/jane/.local/share",
			}, "/home/jane"))
			if err != nil {
				t.Fatalf("ResolveIn() error = %v", err)
			}
			if paths.Root != custom {
				t.Errorf("Root = %q, want the override %q", paths.Root, custom)
			}
		})
	}
}

func TestOverrideIsMadeAbsolute(t *testing.T) {
	paths, err := ResolveIn(envFrom("linux", map[string]string{EnvDataDir: "relative/data"}, "/home/jane"))
	if err != nil {
		t.Fatalf("ResolveIn() error = %v", err)
	}
	if !filepath.IsAbs(paths.Root) {
		t.Errorf("Root = %q, want an absolute path", paths.Root)
	}
	if !strings.HasSuffix(paths.Root, filepath.Join("relative", "data")) {
		t.Errorf("Root = %q, want it to end with the requested directory", paths.Root)
	}
}

// An empty variable is not a choice; it must not shadow the platform default.
func TestEmptyOverrideIsIgnored(t *testing.T) {
	paths, err := ResolveIn(envFrom("linux", map[string]string{
		EnvDataDir:      "",
		"XDG_DATA_HOME": "/home/jane/.local/share",
	}, "/home/jane"))
	if err != nil {
		t.Fatalf("ResolveIn() error = %v", err)
	}
	if want := filepath.Join("/home/jane/.local/share", "echo"); paths.Root != want {
		t.Errorf("Root = %q, want %q", paths.Root, want)
	}
}

func TestResolveFailsActionablyWhenLocationIsUnknown(t *testing.T) {
	tests := []struct {
		name string
		goos string
		vars map[string]string
		home string
	}{
		{name: "windows without LOCALAPPDATA", goos: "windows", home: `C:\Users\jane`},
		{name: "linux without a home directory", goos: "linux"},
		{name: "macos without a home directory", goos: "darwin"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolveIn(envFrom(test.goos, test.vars, test.home))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !errors.Is(err, ErrNoDataDir) {
				t.Errorf("error = %v, want it to wrap ErrNoDataDir", err)
			}
			if !strings.Contains(err.Error(), EnvDataDir) {
				t.Errorf("error = %q, want it to name %s as the way out", err, EnvDataDir)
			}
		})
	}
}

func TestDerivedPathsSitUnderTheRoot(t *testing.T) {
	paths := Paths{Root: filepath.Join("data", "root")}

	if want := filepath.Join(paths.Root, "conversations"); paths.ConversationsDir() != want {
		t.Errorf("ConversationsDir() = %q, want %q", paths.ConversationsDir(), want)
	}
	if want := filepath.Join(paths.Root, "conversations", "abc123"); paths.ConversationDir("abc123") != want {
		t.Errorf("ConversationDir() = %q, want %q", paths.ConversationDir("abc123"), want)
	}
}

// Resolution must read the environment, never a path baked into the binary.
func TestResolveReadsTheHostEnvironment(t *testing.T) {
	custom := t.TempDir()
	t.Setenv(EnvDataDir, custom)

	paths, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if paths.Root != custom {
		t.Errorf("Root = %q, want %q", paths.Root, custom)
	}
}
