// Package config resolves where Echo keeps its data.
//
// Resolution is deterministic and depends only on the operating system and the
// environment, never on a path compiled into the binary. Windows is the primary
// target; the other platforms are supported so the suite and CI can run
// anywhere.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// EnvDataDir overrides the data root. It exists for tests and automation, which
// must never touch a developer's real conversations.
const EnvDataDir = "ECHO_DATA_DIR"

// dirName is the directory Echo creates inside the platform's data location.
// It is the product name, not the command name.
const dirName = "Echo"

// Paths locates everything Echo stores on disk. Construct it with Resolve.
type Paths struct {
	// Root is the data directory containing all Echo state.
	Root string
}

// ConversationsDir holds one directory per conversation.
func (p Paths) ConversationsDir() string {
	return filepath.Join(p.Root, "conversations")
}

// ConversationDir is the workspace for a single conversation.
func (p Paths) ConversationDir(id string) string {
	return filepath.Join(p.ConversationsDir(), id)
}

// Environment is the outside world that resolution reads. Injecting it keeps
// every platform's behavior testable from any host.
type Environment struct {
	// GOOS names the target platform, matching runtime.GOOS.
	GOOS string
	// Getenv reads an environment variable.
	Getenv func(string) string
	// UserHomeDir reports the current user's home directory.
	UserHomeDir func() (string, error)
}

// hostEnvironment reads the real process environment.
func hostEnvironment() Environment {
	return Environment{
		GOOS:        runtime.GOOS,
		Getenv:      os.Getenv,
		UserHomeDir: os.UserHomeDir,
	}
}

// ErrNoDataDir means neither the override nor the platform's conventional
// location could be determined. The wrapped message names what was missing.
var ErrNoDataDir = errors.New("cannot determine a data directory")

// Resolve reports where Echo stores data on this machine.
func Resolve() (Paths, error) {
	return ResolveIn(hostEnvironment())
}

// ResolveIn reports where Echo stores data for the given environment.
//
// The override wins outright. Otherwise the platform's conventional location is
// used: %LOCALAPPDATA%\Echo on Windows, ~/Library/Application Support/Echo on
// macOS, and $XDG_DATA_HOME/echo or ~/.local/share/echo elsewhere.
func ResolveIn(env Environment) (Paths, error) {
	if override := env.Getenv(EnvDataDir); override != "" {
		absolute, err := filepath.Abs(override)
		if err != nil {
			return Paths{}, fmt.Errorf("%s is not a usable path: %w", EnvDataDir, err)
		}

		return Paths{Root: absolute}, nil
	}

	root, err := platformRoot(env)
	if err != nil {
		return Paths{}, err
	}

	return Paths{Root: root}, nil
}

// platformRoot applies the conventional location for env.GOOS.
func platformRoot(env Environment) (string, error) {
	switch env.GOOS {
	case "windows":
		// LOCALAPPDATA is machine-local and excluded from roaming profiles,
		// which is correct for recordings and transcripts.
		if local := env.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, dirName), nil
		}

		return "", fmt.Errorf("%w: LOCALAPPDATA is not set; set %s to choose a location explicitly", ErrNoDataDir, EnvDataDir)

	case "darwin":
		home, err := env.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("%w: %w; set %s to choose a location explicitly", ErrNoDataDir, err, EnvDataDir)
		}

		return filepath.Join(home, "Library", "Application Support", dirName), nil

	default:
		if share := env.Getenv("XDG_DATA_HOME"); share != "" {
			return filepath.Join(share, "echo"), nil
		}

		home, err := env.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("%w: %w; set %s to choose a location explicitly", ErrNoDataDir, err, EnvDataDir)
		}

		return filepath.Join(home, ".local", "share", "echo"), nil
	}
}
