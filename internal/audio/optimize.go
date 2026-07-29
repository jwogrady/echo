package audio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jwogrady/echo/internal/conversation"
)

// convertTimeout bounds an ffmpeg run. Long recordings take a while, so this is
// generous; the point is that a wedged process cannot hang Echo forever.
const convertTimeout = 30 * time.Minute

// ConvertName is the conversion tool Echo invokes.
const ConvertName = "ffmpeg"

// The derivative's format. faster-whisper expects mono 16 kHz signed 16-bit PCM,
// so producing exactly that means the worker never resamples.
const (
	// TargetChannels is mono.
	TargetChannels = 1
	// TargetSampleRate is 16 kHz.
	TargetSampleRate = 16000
	// TargetCodec is signed 16-bit little-endian PCM.
	TargetCodec = "pcm_s16le"
	// TargetSampleFormat is what ffprobe reports for that codec.
	TargetSampleFormat = "s16"
)

// Optimization errors.
var (
	// ErrConvertFailed means ffmpeg ran and reported failure.
	ErrConvertFailed = errors.New("ffmpeg could not convert the audio")
	// ErrOutputMissing means ffmpeg exited zero but produced nothing usable.
	ErrOutputMissing = errors.New("ffmpeg produced no output file")
	// ErrOutputWrongFormat means the derivative is not the format requested. An
	// exit code of zero is not proof the file is usable.
	ErrOutputWrongFormat = errors.New("converted audio is not in the expected format")
)

// Additional pipeline stages for conversion.
const (
	StageOptimizing Stage = "optimizing audio"
	StageValidating Stage = "validating optimized audio"
)

// Converter produces the transcription-ready derivative.
type Converter struct {
	// Runner executes ffmpeg.
	Runner Runner
	// Inspect validates the output. Injected so the check is testable.
	Inspect func(ctx context.Context, path string) (Properties, error)
	// Progress receives stage announcements. Optional.
	Progress func(Stage)
}

// NewConverter returns a Converter using the real tools.
func NewConverter() *Converter {
	prober := NewProber()

	return &Converter{
		Runner:  NewExecRunner(),
		Inspect: prober.Inspect,
	}
}

func (c *Converter) announce(stage Stage) {
	if c.Progress != nil {
		c.Progress(stage)
	}
}

// convertArgs builds ffmpeg's arguments as an array.
//
// No shell is involved anywhere in this path, so a path containing a space, a
// quote, or a semicolon is data rather than syntax. The milestone forbids
// shell-concatenated commands explicitly.
func convertArgs(source, destination string) []string {
	return []string{
		// Fail rather than prompt if the destination somehow exists; Echo
		// controls the temporary name, so a prompt would hang forever.
		"-nostdin",
		"-y",
		"-v", "error",
		"-i", source,
		"-ac", fmt.Sprint(TargetChannels),
		"-ar", fmt.Sprint(TargetSampleRate),
		"-acodec", TargetCodec,
		// Drop any video or subtitle stream rather than failing on it.
		"-vn",
		"-sn",
		"-f", "wav",
		destination,
	}
}

// Optimize produces the derivative for a conversation's imported source.
//
// The source is read and never written. The output is built under a temporary
// name and only renamed into place after it has been re-inspected and confirmed
// to be the requested format — ffmpeg exiting zero is not proof of a usable file.
// A failure at any point leaves no partial derivative behind.
func (c *Converter) Optimize(ctx context.Context, workspace conversation.Workspace) (Properties, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	source := SourcePath(workspace)
	if _, err := os.Stat(source); err != nil {
		return Properties{}, &StageError{
			Stage: StageOptimizing,
			Cause: fmt.Errorf("no imported audio at %s: %w", source, err),
		}
	}

	destination := OptimizedPath(workspace)

	// Convert to a temporary sibling so an interrupted run cannot leave a
	// half-written optimized.wav that later stages would trust.
	temp, err := os.CreateTemp(filepath.Dir(destination), "."+OptimizedName+".tmp-*")
	if err != nil {
		return Properties{}, &StageError{Stage: StageOptimizing, Cause: err}
	}
	tempName := temp.Name()
	// ffmpeg writes the file itself; Echo only needed the unique name.
	_ = temp.Close()

	cleanup := func() { _ = os.Remove(tempName) }

	bounded, cancel := context.WithTimeout(ctx, convertTimeout)
	defer cancel()

	c.announce(StageOptimizing)
	if _, err := c.Runner.Run(bounded, ConvertName, convertArgs(source, tempName)...); err != nil {
		cleanup()

		if errors.Is(err, ErrToolMissing) {
			return Properties{}, &StageError{Stage: StageOptimizing, Cause: err}
		}

		return Properties{}, &StageError{
			Stage: StageOptimizing,
			Cause: fmt.Errorf("%w: %w", ErrConvertFailed, err),
		}
	}

	info, err := os.Stat(tempName)
	if err != nil || info.Size() == 0 {
		cleanup()

		return Properties{}, &StageError{
			Stage: StageOptimizing,
			Cause: fmt.Errorf("%w at %s", ErrOutputMissing, tempName),
		}
	}

	c.announce(StageValidating)
	properties, err := c.Inspect(bounded, tempName)
	if err != nil {
		cleanup()

		return Properties{}, &StageError{Stage: StageValidating, Cause: err}
	}

	if err := checkTargetFormat(properties); err != nil {
		cleanup()

		return Properties{}, &StageError{Stage: StageValidating, Cause: err}
	}

	if err := os.Rename(tempName, destination); err != nil {
		cleanup()

		return Properties{}, &StageError{
			Stage: StageValidating,
			Cause: fmt.Errorf("moving the derivative into place: %w", err),
		}
	}

	return properties, nil
}

// checkTargetFormat confirms the derivative is what was asked for.
func checkTargetFormat(properties Properties) error {
	var problems []error

	if properties.Channels != TargetChannels {
		problems = append(problems, fmt.Errorf("channels is %d, want %d", properties.Channels, TargetChannels))
	}
	if properties.SampleRate != TargetSampleRate {
		problems = append(problems, fmt.Errorf("sample rate is %d, want %d", properties.SampleRate, TargetSampleRate))
	}
	if properties.Codec != TargetCodec {
		problems = append(problems, fmt.Errorf("codec is %q, want %q", properties.Codec, TargetCodec))
	}
	if properties.SampleFormat != TargetSampleFormat {
		problems = append(problems, fmt.Errorf("sample format is %q, want %q", properties.SampleFormat, TargetSampleFormat))
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w: %w", ErrOutputWrongFormat, errors.Join(problems...))
	}

	return nil
}

// EnsureOptimized guarantees a valid derivative exists, building one only if
// needed. It reports whether it had to build.
//
// This is what makes `add` safe to retry. A run interrupted after the copy but
// before conversion leaves a conversation with source audio and no derivative;
// without this, re-adding the same file would report "already imported" and the
// conversation would never become usable.
func (c *Converter) EnsureOptimized(ctx context.Context, workspace conversation.Workspace) (Properties, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	existing := OptimizedPath(workspace)

	info, err := os.Stat(existing)
	switch {
	case err != nil:
		// Nothing there, or unreadable: build it.
	case info.Size() == 0:
		// A zero-length file is debris from an interrupted run.
	default:
		properties, inspectErr := c.Inspect(ctx, existing)
		if inspectErr == nil && checkTargetFormat(properties) == nil {
			return properties, false, nil
		}
		// Present but unusable — a truncated or wrong-format file must be
		// rebuilt rather than trusted.
	}

	properties, err := c.Optimize(ctx, workspace)

	return properties, true, err
}
