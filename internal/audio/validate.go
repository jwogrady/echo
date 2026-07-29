// Package audio inspects, imports, and converts recordings.
//
// Validation here is pure inspection: nothing in this file creates, moves, or
// modifies a file. Deciding whether Echo can handle a recording must never be
// able to damage it.
package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Code is a stable, documented identifier for a validation failure.
//
// These are part of Echo's contract: the troubleshooting documentation maps them
// to remediation, and tests assert on a code rather than on prose so wording can
// improve without breaking anything.
type Code string

const (
	// CodeNotFound means nothing exists at the given path.
	CodeNotFound Code = "audio_not_found"
	// CodeNotRegularFile means the path is a directory, device, or similar.
	CodeNotRegularFile Code = "audio_not_regular_file"
	// CodeEmpty means the file has no contents.
	CodeEmpty Code = "audio_empty"
	// CodeUnreadable means the file exists but cannot be read.
	CodeUnreadable Code = "audio_unreadable"
	// CodeNotWAV means the contents are not a RIFF/WAVE container.
	CodeNotWAV Code = "audio_not_wav"
	// CodeTruncated means the container header is incomplete.
	CodeTruncated Code = "audio_truncated"
	// CodeWrongExtension means the contents are a WAV but the name is not .wav.
	CodeWrongExtension Code = "audio_wrong_extension"
)

// Error is a validation failure carrying a stable code.
type Error struct {
	// Code identifies the failure kind.
	Code Code
	// Path is the file that was rejected.
	Path string
	// Message explains the failure to a person.
	Message string
	// Cause is the underlying error, when one exists.
	Cause error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

// Is lets callers match on a bare code with errors.Is.
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)

	return ok && other.Code == e.Code
}

// CodeOf reports the validation code carried by err, or "" if it carries none.
func CodeOf(err error) Code {
	var validation *Error
	if errors.As(err, &validation) {
		return validation.Code
	}

	return ""
}

// Extension is the only container Echo accepts for v0.1. MP3 and video input are
// explicit non-goals in the project overview.
const Extension = ".wav"

// riffHeaderSize is the number of bytes needed to identify a RIFF/WAVE file:
// "RIFF", a 4-byte size, then "WAVE".
const riffHeaderSize = 12

// Source is a validated input file, ready to import.
type Source struct {
	// Path is the absolute path to the file.
	Path string
	// Name is the original filename, recorded as provenance.
	Name string
	// Size is the file size in bytes.
	Size int64
}

// Validate inspects path and reports whether Echo can import it.
//
// Checks run cheapest-first so an obviously wrong input fails without reading
// the file. Nothing is created, moved, or modified.
func Validate(path string) (Source, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Source{}, &Error{
			Code:    CodeNotFound,
			Path:    path,
			Message: "cannot be resolved to a real path",
			Cause:   err,
		}
	}

	info, err := os.Stat(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Source{}, &Error{
				Code:    CodeNotFound,
				Path:    absolute,
				Message: "does not exist",
				Cause:   err,
			}
		}

		return Source{}, &Error{
			Code:    CodeUnreadable,
			Path:    absolute,
			Message: "cannot be read",
			Cause:   err,
		}
	}

	// Stat follows symlinks, so a link to a real WAV is fine while a link to
	// nothing already failed above.
	if info.IsDir() {
		return Source{}, &Error{
			Code:    CodeNotRegularFile,
			Path:    absolute,
			Message: "is a directory, not a file",
		}
	}
	if !info.Mode().IsRegular() {
		return Source{}, &Error{
			Code:    CodeNotRegularFile,
			Path:    absolute,
			Message: "is not a regular file",
		}
	}
	if info.Size() == 0 {
		return Source{}, &Error{
			Code:    CodeEmpty,
			Path:    absolute,
			Message: "is empty",
		}
	}

	// The contents decide, not the name. A .wav holding an MP3 must be refused.
	if err := checkRIFFWAVE(absolute); err != nil {
		return Source{}, err
	}

	// Only once the contents are known to be a WAV is a wrong extension worth
	// mentioning, so the message can be specific rather than guessing.
	if !strings.EqualFold(filepath.Ext(absolute), Extension) {
		return Source{}, &Error{
			Code: CodeWrongExtension,
			Path: absolute,
			Message: fmt.Sprintf("contains WAV audio but is not named %s; rename it so its name matches its contents",
				Extension),
		}
	}

	return Source{
		Path: absolute,
		Name: filepath.Base(absolute),
		Size: info.Size(),
	}, nil
}

// checkRIFFWAVE confirms the file begins with a RIFF/WAVE container header.
func checkRIFFWAVE(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return &Error{
			Code:    CodeUnreadable,
			Path:    path,
			Message: "cannot be opened for reading",
			Cause:   err,
		}
	}
	defer func() { _ = file.Close() }()

	header := make([]byte, riffHeaderSize)
	if _, err := io.ReadFull(file, header); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return &Error{
				Code:    CodeTruncated,
				Path:    path,
				Message: fmt.Sprintf("is too small to be a WAV file; a container header needs %d bytes", riffHeaderSize),
				Cause:   err,
			}
		}

		return &Error{
			Code:    CodeUnreadable,
			Path:    path,
			Message: "cannot be read",
			Cause:   err,
		}
	}

	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return &Error{
			Code: CodeNotWAV,
			Path: path,
			Message: fmt.Sprintf("is not a WAV file; its contents begin with %q, not a RIFF/WAVE header. Echo v0.1 accepts WAV only",
				printable(header[0:4])),
		}
	}

	// The declared size is advisory — many writers leave it wrong on a stream —
	// so a mismatch is not a rejection. Reading it here documents that choice.
	_ = binary.LittleEndian.Uint32(header[4:8])

	return nil
}

// printable renders header bytes for a message without emitting control
// characters into a terminal.
func printable(b []byte) string {
	var builder strings.Builder
	for _, c := range b {
		if c >= 0x20 && c < 0x7f {
			builder.WriteByte(c)
			continue
		}
		builder.WriteString(fmt.Sprintf("\\x%02x", c))
	}

	return builder.String()
}
