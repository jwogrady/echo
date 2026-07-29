package audio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// probeTimeout bounds an ffprobe invocation. A hung tool must not hang Echo.
const probeTimeout = 30 * time.Second

// Probe errors distinguishable by callers.
var (
	// ErrToolMissing means ffprobe or ffmpeg is not installed.
	ErrToolMissing = errors.New("required tool is not available")
	// ErrProbeFailed means the tool ran and reported failure.
	ErrProbeFailed = errors.New("ffprobe could not read the file")
	// ErrProbeOutput means the tool's output was not the expected shape. A
	// missing or unparseable field is this, never a zero value: a duration of 0
	// because parsing failed would later look like an empty recording.
	ErrProbeOutput = errors.New("ffprobe output could not be parsed")
	// ErrNoAudioStream means the file contains no audio.
	ErrNoAudioStream = errors.New("file contains no audio stream")
)

// Runner executes an external tool. Injecting it makes every parse path testable
// without the binary installed.
type Runner interface {
	// Run executes name with args and returns its stdout. Arguments are passed as
	// an array, never a shell string.
	Run(ctx context.Context, name string, args ...string) (stdout []byte, err error)
}

// execRunner runs real processes.
type execRunner struct{}

// NewExecRunner returns a Runner backed by os/exec.
func NewExecRunner() Runner { return execRunner{} }

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("%w: %s is not on PATH", ErrToolMissing, name)
	}

	// CommandContext takes an argument array. No shell is involved, so a path
	// containing a space, quote, or semicolon is data rather than syntax.
	command := exec.CommandContext(ctx, name, args...)

	stdout, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("%w: %s exited %d: %s",
				ErrProbeFailed, name, exitErr.ExitCode(), truncate(string(exitErr.Stderr), 400))
		}

		return nil, fmt.Errorf("running %s: %w", name, err)
	}

	return stdout, nil
}

// truncate bounds tool output so a runaway stderr cannot fill a terminal or a
// job record.
func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	return s[:limit] + "… (truncated)"
}

// ProbeName is the inspection tool Echo invokes.
const ProbeName = "ffprobe"

// Prober inspects media files with ffprobe.
type Prober struct {
	Runner Runner
}

// NewProber returns a Prober that runs the real ffprobe.
func NewProber() *Prober {
	return &Prober{Runner: NewExecRunner()}
}

// probeDocument mirrors the subset of ffprobe's JSON that Echo reads.
//
// Numbers arrive as strings in ffprobe's output, so they are parsed explicitly
// rather than trusted to unmarshal.
type probeDocument struct {
	Streams []struct {
		CodecName  string `json:"codec_name"`
		CodecType  string `json:"codec_type"`
		SampleFmt  string `json:"sample_fmt"`
		SampleRate string `json:"sample_rate"`
		Channels   int    `json:"channels"`
		Duration   string `json:"duration"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
	} `json:"format"`
}

// Inspect reports the audio properties of the file at path.
//
// Parsing is strict: a missing field, an unparseable number, or an absent audio
// stream is an error naming what was wrong. Nothing is defaulted.
func (p *Prober) Inspect(ctx context.Context, path string) (Properties, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	bounded, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	stdout, err := p.Runner.Run(bounded, ProbeName,
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		"-select_streams", "a:0",
		path,
	)
	if err != nil {
		return Properties{}, err
	}

	var document probeDocument
	if err := json.Unmarshal(stdout, &document); err != nil {
		return Properties{}, fmt.Errorf("%w: %s: %w", ErrProbeOutput, path, err)
	}

	if len(document.Streams) == 0 {
		return Properties{}, fmt.Errorf("%w: %s", ErrNoAudioStream, path)
	}

	stream := document.Streams[0]
	if stream.CodecType != "audio" {
		return Properties{}, fmt.Errorf("%w: %s: first stream is %q, not audio",
			ErrNoAudioStream, path, stream.CodecType)
	}

	properties := Properties{
		Codec:        stream.CodecName,
		SampleFormat: stream.SampleFmt,
		Channels:     stream.Channels,
	}

	if properties.Codec == "" {
		return Properties{}, fmt.Errorf("%w: %s: codec_name is missing", ErrProbeOutput, path)
	}
	if properties.Channels <= 0 {
		return Properties{}, fmt.Errorf("%w: %s: channels is %d", ErrProbeOutput, path, properties.Channels)
	}

	sampleRate, err := strconv.Atoi(stream.SampleRate)
	if err != nil || sampleRate <= 0 {
		return Properties{}, fmt.Errorf("%w: %s: sample_rate %q is not a positive integer",
			ErrProbeOutput, path, stream.SampleRate)
	}
	properties.SampleRate = sampleRate

	// Duration may be reported on the stream, the container, or neither. WAV
	// written without a correct size field genuinely has no duration to report,
	// so the two sources are tried in order and absence is an error rather than a
	// silent zero.
	duration, err := firstDuration(stream.Duration, document.Format.Duration)
	if err != nil {
		return Properties{}, fmt.Errorf("%w: %s: %w", ErrProbeOutput, path, err)
	}
	properties.DurationSeconds = duration

	return properties, nil
}

// firstDuration parses the first usable duration from the candidates.
func firstDuration(candidates ...string) (float64, error) {
	for _, candidate := range candidates {
		if candidate == "" || candidate == "N/A" {
			continue
		}

		seconds, err := strconv.ParseFloat(candidate, 64)
		if err != nil {
			return 0, fmt.Errorf("duration %q is not a number", candidate)
		}
		if seconds < 0 {
			return 0, fmt.Errorf("duration %q is negative", candidate)
		}

		return seconds, nil
	}

	return 0, errors.New("no duration reported on the stream or the container")
}
