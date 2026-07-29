package audio

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwogrady/echo/internal/conversation"
)

// scriptedRunner writes a chosen file when invoked, so conversion outcomes can be
// simulated without ffmpeg.
type scriptedRunner struct {
	// onRun receives the destination path (ffmpeg's last argument).
	onRun func(destination string) error
	calls [][]string
}

func (s *scriptedRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	s.calls = append(s.calls, append([]string{name}, args...))

	if s.onRun == nil {
		return nil, nil
	}

	return nil, s.onRun(args[len(args)-1])
}

// targetProperties is what a correct derivative looks like.
func targetProperties() Properties {
	return Properties{
		Codec:           TargetCodec,
		SampleFormat:    TargetSampleFormat,
		SampleRate:      TargetSampleRate,
		Channels:        TargetChannels,
		DurationSeconds: 2,
	}
}

// workspaceWithSource returns a workspace holding an imported source.wav.
func workspaceWithSource(t *testing.T) conversation.Workspace {
	t.Helper()

	workspace := newTestWorkspace(t)
	if err := os.WriteFile(SourcePath(workspace), wavBytes(silence()), 0o644); err != nil {
		t.Fatalf("writing the source: %v", err)
	}

	return workspace
}

func TestOptimizeProducesAValidatedDerivative(t *testing.T) {
	workspace := workspaceWithSource(t)

	runner := &scriptedRunner{onRun: func(destination string) error {
		return os.WriteFile(destination, wavBytes(silence()), 0o644)
	}}
	converter := &Converter{
		Runner:  runner,
		Inspect: func(context.Context, string) (Properties, error) { return targetProperties(), nil },
	}

	properties, err := converter.Optimize(context.Background(), workspace)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if properties.SampleRate != TargetSampleRate || properties.Channels != TargetChannels {
		t.Errorf("properties = %+v", properties)
	}
	if _, err := os.Stat(OptimizedPath(workspace)); err != nil {
		t.Errorf("the derivative is not in place: %v", err)
	}
}

// The milestone forbids shell-concatenated commands.
func TestConvertArgsRequestTheTargetFormatSafely(t *testing.T) {
	source := `/tmp/a dir; rm -rf $HOME/"quoted".wav`
	destination := "/tmp/out dir/optimized.wav"

	args := convertArgs(source, destination)

	if args[len(args)-1] != destination {
		t.Errorf("last argument = %q, want the destination", args[len(args)-1])
	}

	var sawSource bool
	for _, arg := range args {
		if arg == source {
			sawSource = true
		}
		if strings.Contains(arg, "&&") || strings.Contains(arg, "|") || strings.Contains(arg, ">") {
			t.Errorf("argument %q looks like shell syntax", arg)
		}
	}
	if !sawSource {
		t.Error("the source path is not passed as its own argument")
	}

	joined := strings.Join(args, " ")
	for _, want := range []string{"-ac 1", "-ar 16000", "-acodec pcm_s16le", "-f wav", "-nostdin"} {
		if !strings.Contains(joined, want) {
			t.Errorf("arguments missing %q: %v", want, args)
		}
	}
}

// ffmpeg exiting zero is not proof the output is usable.
func TestOptimizeRejectsAWrongFormatOutput(t *testing.T) {
	tests := []struct {
		name       string
		properties Properties
		wantIn     string
	}{
		{
			name: "still stereo",
			properties: func() Properties {
				p := targetProperties()
				p.Channels = 2
				return p
			}(),
			wantIn: "channels is 2",
		},
		{
			name: "wrong sample rate",
			properties: func() Properties {
				p := targetProperties()
				p.SampleRate = 44100
				return p
			}(),
			wantIn: "sample rate is 44100",
		},
		{
			name: "wrong codec",
			properties: func() Properties {
				p := targetProperties()
				p.Codec = "mp3"
				return p
			}(),
			wantIn: `codec is "mp3"`,
		},
		{
			name: "wrong sample format",
			properties: func() Properties {
				p := targetProperties()
				p.SampleFormat = "flt"
				return p
			}(),
			wantIn: `sample format is "flt"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := workspaceWithSource(t)

			converter := &Converter{
				Runner: &scriptedRunner{onRun: func(destination string) error {
					return os.WriteFile(destination, wavBytes(silence()), 0o644)
				}},
				Inspect: func(context.Context, string) (Properties, error) { return test.properties, nil },
			}

			_, err := converter.Optimize(context.Background(), workspace)
			if !errors.Is(err, ErrOutputWrongFormat) {
				t.Fatalf("error = %v, want ErrOutputWrongFormat", err)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("error = %q, want it to name the mismatch %q", err, test.wantIn)
			}

			// A rejected derivative must not be left in place.
			if _, statErr := os.Stat(OptimizedPath(workspace)); !errors.Is(statErr, os.ErrNotExist) {
				t.Error("a wrong-format derivative was left in place")
			}
		})
	}
}

// Every failure path must leave no partial derivative and no temporary file.
func TestOptimizeCleansUpAfterFailure(t *testing.T) {
	tests := []struct {
		name    string
		runner  *scriptedRunner
		inspect func(context.Context, string) (Properties, error)
		wantErr error
	}{
		{
			name:    "ffmpeg fails",
			runner:  &scriptedRunner{onRun: func(string) error { return errors.New("exit 1") }},
			inspect: func(context.Context, string) (Properties, error) { return targetProperties(), nil },
			wantErr: ErrConvertFailed,
		},
		{
			name:    "ffmpeg missing",
			runner:  &scriptedRunner{onRun: func(string) error { return ErrToolMissing }},
			inspect: func(context.Context, string) (Properties, error) { return targetProperties(), nil },
			wantErr: ErrToolMissing,
		},
		{
			name:    "exits zero but writes nothing",
			runner:  &scriptedRunner{onRun: func(string) error { return nil }},
			inspect: func(context.Context, string) (Properties, error) { return targetProperties(), nil },
			wantErr: ErrOutputMissing,
		},
		{
			name: "output cannot be inspected",
			runner: &scriptedRunner{onRun: func(destination string) error {
				return os.WriteFile(destination, []byte("junk"), 0o644)
			}},
			inspect: func(context.Context, string) (Properties, error) { return Properties{}, ErrProbeFailed },
			wantErr: ErrProbeFailed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := workspaceWithSource(t)

			converter := &Converter{Runner: test.runner, Inspect: test.inspect}

			_, err := converter.Optimize(context.Background(), workspace)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}

			var staged *StageError
			if !errors.As(err, &staged) {
				t.Errorf("error = %v, want it to name a stage", err)
			}

			entries, readErr := os.ReadDir(workspace.AudioPath())
			if readErr != nil {
				t.Fatalf("reading the audio directory: %v", readErr)
			}
			for _, entry := range entries {
				if entry.Name() != SourceName {
					t.Errorf("left behind %s after a failure", entry.Name())
				}
			}
		})
	}
}

func TestOptimizeRequiresAnImportedSource(t *testing.T) {
	workspace := newTestWorkspace(t)

	converter := &Converter{
		Runner:  &scriptedRunner{},
		Inspect: func(context.Context, string) (Properties, error) { return targetProperties(), nil },
	}

	_, err := converter.Optimize(context.Background(), workspace)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "no imported audio") {
		t.Errorf("error = %q, want it to say there is nothing to convert", err)
	}
}

// The real thing: convert a genuine stereo 44.1 kHz file and re-probe the result.
func TestOptimizeWithRealFFmpeg(t *testing.T) {
	for _, tool := range []string{ConvertName, ProbeName} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}

	workspace := newTestWorkspace(t)

	// Build a stereo 44.1 kHz source with ffmpeg so the conversion has real work.
	source := SourcePath(workspace)
	build := exec.Command(ConvertName, "-nostdin", "-v", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-ac", "2", "-ar", "44100", "-y", source)
	if output, err := build.CombinedOutput(); err != nil {
		t.Skipf("could not build a source fixture: %v: %s", err, output)
	}

	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("reading the source: %v", err)
	}

	properties, err := NewConverter().Optimize(context.Background(), workspace)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}

	if properties.Channels != TargetChannels {
		t.Errorf("Channels = %d, want %d", properties.Channels, TargetChannels)
	}
	if properties.SampleRate != TargetSampleRate {
		t.Errorf("SampleRate = %d, want %d", properties.SampleRate, TargetSampleRate)
	}
	if properties.Codec != TargetCodec {
		t.Errorf("Codec = %q, want %q", properties.Codec, TargetCodec)
	}

	// The source must be untouched by conversion.
	after, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("reading the source back: %v", err)
	}
	if string(before) != string(after) {
		t.Error("conversion modified the source audio")
	}

	// And no temporary files survive a success.
	entries, err := os.ReadDir(workspace.AudioPath())
	if err != nil {
		t.Fatalf("reading the audio directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("left a temporary file behind: %s", entry.Name())
		}
	}
}

// A path with spaces and shell metacharacters must survive real ffmpeg.
func TestRealFFmpegHandlesAwkwardPaths(t *testing.T) {
	for _, tool := range []string{ConvertName, ProbeName} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}

	awkward := filepath.Join(t.TempDir(), "a dir with spaces; and $vars")
	workspace := conversation.NewWorkspace(awkward)
	if err := os.MkdirAll(workspace.AudioPath(), 0o755); err != nil {
		t.Fatalf("creating the workspace: %v", err)
	}
	if err := os.WriteFile(SourcePath(workspace), wavBytes(silence()), 0o644); err != nil {
		t.Fatalf("writing the source: %v", err)
	}

	if _, err := NewConverter().Optimize(context.Background(), workspace); err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if _, err := os.Stat(OptimizedPath(workspace)); err != nil {
		t.Errorf("no derivative produced: %v", err)
	}
}

// EnsureOptimized is what makes `add` safe to retry.
func TestEnsureOptimizedSkipsAValidDerivative(t *testing.T) {
	workspace := workspaceWithSource(t)
	if err := os.WriteFile(OptimizedPath(workspace), wavBytes(silence()), 0o644); err != nil {
		t.Fatalf("writing the derivative: %v", err)
	}

	runner := &scriptedRunner{onRun: func(string) error {
		t.Error("ffmpeg was invoked despite a valid derivative")
		return nil
	}}
	converter := &Converter{
		Runner:  runner,
		Inspect: func(context.Context, string) (Properties, error) { return targetProperties(), nil },
	}

	_, rebuilt, err := converter.EnsureOptimized(context.Background(), workspace)
	if err != nil {
		t.Fatalf("EnsureOptimized() error = %v", err)
	}
	if rebuilt {
		t.Error("rebuilt = true, want false for an already-valid derivative")
	}
}

// A run interrupted after the copy leaves no derivative; retrying must heal it.
func TestEnsureOptimizedRebuildsWhatIsMissingOrUnusable(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, workspace conversation.Workspace)
		valid bool
	}{
		{
			name:  "no derivative at all",
			setup: func(*testing.T, conversation.Workspace) {},
			valid: true,
		},
		{
			name: "zero-length debris",
			setup: func(t *testing.T, workspace conversation.Workspace) {
				if err := os.WriteFile(OptimizedPath(workspace), nil, 0o644); err != nil {
					t.Fatalf("writing debris: %v", err)
				}
			},
			valid: true,
		},
		{
			name: "unprobeable file",
			setup: func(t *testing.T, workspace conversation.Workspace) {
				if err := os.WriteFile(OptimizedPath(workspace), []byte("RIFFtruncated"), 0o644); err != nil {
					t.Fatalf("writing debris: %v", err)
				}
			},
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := workspaceWithSource(t)
			test.setup(t, workspace)

			var invoked bool
			// The existing file is unusable; only the freshly written one probes clean.
			fresh := false
			converter := &Converter{
				Runner: &scriptedRunner{onRun: func(destination string) error {
					invoked = true
					fresh = true
					return os.WriteFile(destination, wavBytes(silence()), 0o644)
				}},
				Inspect: func(_ context.Context, path string) (Properties, error) {
					if !fresh && !test.valid {
						return Properties{}, ErrProbeFailed
					}
					return targetProperties(), nil
				},
			}

			_, rebuilt, err := converter.EnsureOptimized(context.Background(), workspace)
			if err != nil {
				t.Fatalf("EnsureOptimized() error = %v", err)
			}
			if !rebuilt || !invoked {
				t.Error("the derivative should have been rebuilt")
			}
			if _, err := os.Stat(OptimizedPath(workspace)); err != nil {
				t.Errorf("no derivative in place afterward: %v", err)
			}
		})
	}
}

// A wrong-format derivative must be rebuilt rather than accepted.
func TestEnsureOptimizedRebuildsAWrongFormatDerivative(t *testing.T) {
	workspace := workspaceWithSource(t)
	if err := os.WriteFile(OptimizedPath(workspace), wavBytes(silence()), 0o644); err != nil {
		t.Fatalf("writing the derivative: %v", err)
	}

	rebuiltOnce := false
	converter := &Converter{
		Runner: &scriptedRunner{onRun: func(destination string) error {
			rebuiltOnce = true
			return os.WriteFile(destination, wavBytes(silence()), 0o644)
		}},
		Inspect: func(context.Context, string) (Properties, error) {
			if !rebuiltOnce {
				stereo := targetProperties()
				stereo.Channels = 2

				return stereo, nil
			}

			return targetProperties(), nil
		},
	}

	_, rebuilt, err := converter.EnsureOptimized(context.Background(), workspace)
	if err != nil {
		t.Fatalf("EnsureOptimized() error = %v", err)
	}
	if !rebuilt {
		t.Error("a stereo derivative should have been rebuilt")
	}
}
