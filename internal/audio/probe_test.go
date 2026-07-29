package audio

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// fakeRunner returns canned output, so every parse path is testable without
// ffprobe installed.
type fakeRunner struct {
	stdout string
	err    error
	// calls records the arguments each invocation received.
	calls [][]string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))

	return []byte(f.stdout), f.err
}

// probeJSON is a realistic ffprobe document for a mono 16 kHz PCM WAV.
const probeJSON = `{
  "streams": [{
    "codec_name": "pcm_s16le",
    "codec_type": "audio",
    "sample_fmt": "s16",
    "sample_rate": "16000",
    "channels": 1,
    "duration": "2.000000"
  }],
  "format": {"format_name": "wav", "duration": "2.000000"}
}`

func TestInspectParsesRealisticOutput(t *testing.T) {
	runner := &fakeRunner{stdout: probeJSON}

	properties, err := (&Prober{Runner: runner}).Inspect(context.Background(), "/tmp/x.wav")
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if properties.Codec != "pcm_s16le" {
		t.Errorf("Codec = %q", properties.Codec)
	}
	if properties.SampleFormat != "s16" {
		t.Errorf("SampleFormat = %q", properties.SampleFormat)
	}
	if properties.SampleRate != 16000 {
		t.Errorf("SampleRate = %d", properties.SampleRate)
	}
	if properties.Channels != 1 {
		t.Errorf("Channels = %d", properties.Channels)
	}
	if properties.DurationSeconds != 2 {
		t.Errorf("DurationSeconds = %v", properties.DurationSeconds)
	}
}

// Arguments must be an array with no shell involved, and JSON output requested.
func TestInspectPassesArgumentsSafely(t *testing.T) {
	runner := &fakeRunner{stdout: probeJSON}
	path := `/tmp/a dir; rm -rf $HOME/"quoted".wav`

	if _, err := (&Prober{Runner: runner}).Inspect(context.Background(), path); err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("got %d invocations, want 1", len(runner.calls))
	}
	call := runner.calls[0]

	if call[0] != ProbeName {
		t.Errorf("invoked %q, want %q", call[0], ProbeName)
	}
	// The hostile path must arrive as exactly one argument, unmodified.
	if call[len(call)-1] != path {
		t.Errorf("path argument = %q, want %q", call[len(call)-1], path)
	}
	for _, arg := range call {
		if strings.Contains(arg, "&&") || strings.Contains(arg, "|") {
			t.Errorf("argument %q looks like shell syntax", arg)
		}
	}
	joined := strings.Join(call, " ")
	for _, want := range []string{"-print_format json", "-show_streams", "-v error"} {
		if !strings.Contains(joined, want) {
			t.Errorf("invocation missing %q: %v", want, call)
		}
	}
}

// A missing or unparseable field must never become a zero value.
func TestInspectRejectsUnusableOutput(t *testing.T) {
	tests := []struct {
		name    string
		stdout  string
		wantErr error
	}{
		{"not json", "this is not json", ErrProbeOutput},
		{"empty output", "", ErrProbeOutput},
		{"no streams", `{"streams": [], "format": {}}`, ErrNoAudioStream},
		{
			name:    "video stream",
			stdout:  `{"streams":[{"codec_type":"video","codec_name":"h264","sample_rate":"0","channels":0}]}`,
			wantErr: ErrNoAudioStream,
		},
		{
			name:    "missing codec",
			stdout:  `{"streams":[{"codec_type":"audio","sample_rate":"16000","channels":1,"duration":"1.0"}]}`,
			wantErr: ErrProbeOutput,
		},
		{
			name:    "zero channels",
			stdout:  `{"streams":[{"codec_type":"audio","codec_name":"pcm_s16le","sample_rate":"16000","channels":0,"duration":"1.0"}]}`,
			wantErr: ErrProbeOutput,
		},
		{
			name:    "unparseable sample rate",
			stdout:  `{"streams":[{"codec_type":"audio","codec_name":"pcm_s16le","sample_rate":"N/A","channels":1,"duration":"1.0"}]}`,
			wantErr: ErrProbeOutput,
		},
		{
			name:    "zero sample rate",
			stdout:  `{"streams":[{"codec_type":"audio","codec_name":"pcm_s16le","sample_rate":"0","channels":1,"duration":"1.0"}]}`,
			wantErr: ErrProbeOutput,
		},
		{
			name:    "no duration anywhere",
			stdout:  `{"streams":[{"codec_type":"audio","codec_name":"pcm_s16le","sample_rate":"16000","channels":1}],"format":{}}`,
			wantErr: ErrProbeOutput,
		},
		{
			name:    "negative duration",
			stdout:  `{"streams":[{"codec_type":"audio","codec_name":"pcm_s16le","sample_rate":"16000","channels":1,"duration":"-1"}]}`,
			wantErr: ErrProbeOutput,
		},
		{
			name:    "unparseable duration",
			stdout:  `{"streams":[{"codec_type":"audio","codec_name":"pcm_s16le","sample_rate":"16000","channels":1,"duration":"soon"}]}`,
			wantErr: ErrProbeOutput,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := (&Prober{Runner: &fakeRunner{stdout: test.stdout}}).Inspect(context.Background(), "/tmp/x.wav")
			if !errors.Is(err, test.wantErr) {
				t.Errorf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

// Duration on the container but not the stream is normal for some WAVs.
func TestInspectFallsBackToContainerDuration(t *testing.T) {
	stdout := `{"streams":[{"codec_type":"audio","codec_name":"pcm_s16le","sample_rate":"16000","channels":1}],
	            "format":{"duration":"3.5"}}`

	properties, err := (&Prober{Runner: &fakeRunner{stdout: stdout}}).Inspect(context.Background(), "/tmp/x.wav")
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if properties.DurationSeconds != 3.5 {
		t.Errorf("DurationSeconds = %v, want 3.5", properties.DurationSeconds)
	}
}

// "N/A" is ffprobe's way of saying it does not know, not a number.
func TestInspectSkipsNotAvailableDuration(t *testing.T) {
	stdout := `{"streams":[{"codec_type":"audio","codec_name":"pcm_s16le","sample_rate":"16000","channels":1,"duration":"N/A"}],
	            "format":{"duration":"4.25"}}`

	properties, err := (&Prober{Runner: &fakeRunner{stdout: stdout}}).Inspect(context.Background(), "/tmp/x.wav")
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if properties.DurationSeconds != 4.25 {
		t.Errorf("DurationSeconds = %v, want 4.25", properties.DurationSeconds)
	}
}

func TestInspectSurfacesToolFailures(t *testing.T) {
	runner := &fakeRunner{err: ErrProbeFailed}

	if _, err := (&Prober{Runner: runner}).Inspect(context.Background(), "/tmp/x.wav"); !errors.Is(err, ErrProbeFailed) {
		t.Errorf("error = %v, want ErrProbeFailed", err)
	}
}

func TestInspectSurfacesAMissingTool(t *testing.T) {
	runner := &fakeRunner{err: ErrToolMissing}

	if _, err := (&Prober{Runner: runner}).Inspect(context.Background(), "/tmp/x.wav"); !errors.Is(err, ErrToolMissing) {
		t.Errorf("error = %v, want ErrToolMissing", err)
	}
}

func TestTruncateBoundsToolOutput(t *testing.T) {
	if got := truncate("short", 100); got != "short" {
		t.Errorf("truncate() = %q", got)
	}

	long := strings.Repeat("x", 500)
	got := truncate(long, 400)
	if len(got) >= len(long) {
		t.Errorf("truncate() did not shorten a %d-character string", len(long))
	}
	if !strings.HasSuffix(got, "(truncated)") {
		t.Errorf("truncate() = %q, want it to say it truncated", got[len(got)-30:])
	}
}

// The parser is checked against real ffprobe when it is present, so it is
// validated against actual output rather than only fixtures I wrote.
func TestInspectAgainstRealFFprobe(t *testing.T) {
	if _, err := exec.LookPath(ProbeName); err != nil {
		t.Skipf("%s not installed", ProbeName)
	}

	path := writeFixture(t, "real.wav", wavBytes(silence()))

	properties, err := NewProber().Inspect(context.Background(), path)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if properties.Codec != "pcm_s16le" {
		t.Errorf("Codec = %q, want pcm_s16le", properties.Codec)
	}
	if properties.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", properties.SampleRate)
	}
	if properties.Channels != 1 {
		t.Errorf("Channels = %d, want 1", properties.Channels)
	}
	// One second of 16 kHz 16-bit mono.
	if properties.DurationSeconds < 0.9 || properties.DurationSeconds > 1.1 {
		t.Errorf("DurationSeconds = %v, want about 1", properties.DurationSeconds)
	}
}

// Real ffprobe must reject a file that is not audio, exercising the live failure
// path rather than only the faked one.
func TestRealFFprobeRejectsGarbage(t *testing.T) {
	if _, err := exec.LookPath(ProbeName); err != nil {
		t.Skipf("%s not installed", ProbeName)
	}

	path := writeFixture(t, "garbage.wav", []byte(strings.Repeat("not audio", 200)))

	_, err := NewProber().Inspect(context.Background(), path)
	if err == nil {
		t.Fatal("expected an error for a non-audio file")
	}
	if !errors.Is(err, ErrProbeFailed) && !errors.Is(err, ErrNoAudioStream) && !errors.Is(err, ErrProbeOutput) {
		t.Errorf("error = %v, want a probe failure", err)
	}
}
