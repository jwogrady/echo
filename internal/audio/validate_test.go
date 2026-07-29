package audio

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wavBytes builds a minimal but genuine RIFF/WAVE file: a 44-byte canonical
// header plus payload. Generating fixtures keeps audio out of the repository.
func wavBytes(payload []byte) []byte {
	const (
		channels      = 1
		sampleRate    = 16000
		bitsPerSample = 16
	)
	byteRate := sampleRate * channels * bitsPerSample / 8

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+len(payload)))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(header[22:24], channels)
	binary.LittleEndian.PutUint32(header[24:28], sampleRate)
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:34], channels*bitsPerSample/8)
	binary.LittleEndian.PutUint16(header[34:36], bitsPerSample)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(len(payload)))

	return append(header, payload...)
}

// writeFixture writes contents to a named file in a fresh temporary directory.
func writeFixture(t *testing.T, name string, contents []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	return path
}

// silence is a second of 16 kHz mono silence.
func silence() []byte { return make([]byte, 16000*2) }

func TestValidateAcceptsARealWAV(t *testing.T) {
	path := writeFixture(t, "recording.wav", wavBytes(silence()))

	source, err := Validate(path)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if source.Path != path {
		t.Errorf("Path = %q, want %q", source.Path, path)
	}
	if source.Name != "recording.wav" {
		t.Errorf("Name = %q, want %q", source.Name, "recording.wav")
	}
	if want := int64(44 + len(silence())); source.Size != want {
		t.Errorf("Size = %d, want %d", source.Size, want)
	}
}

func TestValidateAcceptsAnUppercaseExtension(t *testing.T) {
	path := writeFixture(t, "RECORDING.WAV", wavBytes(silence()))

	if _, err := Validate(path); err != nil {
		t.Errorf("Validate() error = %v; extension matching should be case-insensitive", err)
	}
}

func TestValidateResolvesToAnAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "recording.wav")
	if err := os.WriteFile(path, wavBytes(silence()), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	t.Chdir(dir)

	source, err := Validate("recording.wav")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !filepath.IsAbs(source.Path) {
		t.Errorf("Path = %q, want an absolute path", source.Path)
	}
}

func TestValidateRejectsWithStableCodes(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		wantCode Code
	}{
		{
			name:     "missing file",
			setup:    func(t *testing.T) string { return filepath.Join(t.TempDir(), "absent.wav") },
			wantCode: CodeNotFound,
		},
		{
			name:     "directory",
			setup:    func(t *testing.T) string { return t.TempDir() },
			wantCode: CodeNotRegularFile,
		},
		{
			name:     "empty file",
			setup:    func(t *testing.T) string { return writeFixture(t, "empty.wav", nil) },
			wantCode: CodeEmpty,
		},
		{
			name:     "too small for a header",
			setup:    func(t *testing.T) string { return writeFixture(t, "tiny.wav", []byte("RIFF")) },
			wantCode: CodeTruncated,
		},
		{
			name: "mp3 wearing a wav extension",
			setup: func(t *testing.T) string {
				return writeFixture(t, "sneaky.wav", []byte("ID3\x04\x00\x00\x00\x00\x00\x00padding"))
			},
			wantCode: CodeNotWAV,
		},
		{
			name: "RIFF but not WAVE",
			setup: func(t *testing.T) string {
				contents := wavBytes(silence())
				copy(contents[8:12], "AVI ")
				return writeFixture(t, "video.wav", contents)
			},
			wantCode: CodeNotWAV,
		},
		{
			name: "real wav named something else",
			setup: func(t *testing.T) string {
				return writeFixture(t, "recording.mp3", wavBytes(silence()))
			},
			wantCode: CodeWrongExtension,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := test.setup(t)

			_, err := Validate(path)
			if err == nil {
				t.Fatal("expected an error")
			}
			if got := CodeOf(err); got != test.wantCode {
				t.Errorf("code = %q, want %q (error: %v)", got, test.wantCode, err)
			}
			if !errors.Is(err, &Error{Code: test.wantCode}) {
				t.Errorf("errors.Is did not match code %q", test.wantCode)
			}
		})
	}
}

// The rejection message must explain which rule was broken, since "invalid file"
// leaves a user with nowhere to go.
func TestRejectionMessagesNameTheProblem(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantSub string
	}{
		{
			name:    "not a wav",
			path:    writeFixture(t, "sneaky.wav", []byte("ID3\x04\x00\x00\x00\x00\x00\x00padding")),
			wantSub: "WAV only",
		},
		{
			name:    "wrong extension",
			path:    writeFixture(t, "recording.mp3", wavBytes(silence())),
			wantSub: "contains WAV audio but is not named",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Validate(test.path)
			if err == nil {
				t.Fatal("expected an error")
			}
			if got := err.Error(); !strings.Contains(got, test.wantSub) {
				t.Errorf("message = %q, want it to contain %q", got, test.wantSub)
			}
			if !strings.Contains(err.Error(), test.path) {
				t.Errorf("message = %q, want it to name the path", err)
			}
		})
	}
}

// Validation must be pure: deciding whether Echo can read a file cannot damage it.
func TestValidateNeverModifiesTheInput(t *testing.T) {
	contents := wavBytes(silence())
	path := writeFixture(t, "recording.wav", contents)

	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}

	if _, err := Validate(path); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Error("Validate changed the modification time")
	}
	if before.Size() != after.Size() {
		t.Error("Validate changed the size")
	}

	readBack, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(readBack) != string(contents) {
		t.Error("Validate changed the contents")
	}
}

// A rejected file must also be left alone, including one Echo refused outright.
func TestValidateLeavesRejectedFilesAlone(t *testing.T) {
	contents := []byte("ID3\x04\x00\x00\x00\x00\x00\x00padding")
	path := writeFixture(t, "sneaky.wav", contents)

	if _, err := Validate(path); err == nil {
		t.Fatal("expected a rejection")
	}

	readBack, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(readBack) != string(contents) {
		t.Error("Validate modified a file it rejected")
	}
}

// A symlink to a real WAV is usable; a broken one is not found.
func TestValidateFollowsSymlinks(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.wav")
	if err := os.WriteFile(real, wavBytes(silence()), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	link := filepath.Join(dir, "link.wav")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Validate(link); err != nil {
		t.Errorf("Validate(symlink) error = %v", err)
	}

	broken := filepath.Join(dir, "broken.wav")
	if err := os.Symlink(filepath.Join(dir, "gone.wav"), broken); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if got := CodeOf(mustFail(t, broken)); got != CodeNotFound {
		t.Errorf("code = %q, want %q", got, CodeNotFound)
	}
}

// Every code must be distinct, or the troubleshooting matrix cannot map them.
func TestCodesAreDistinct(t *testing.T) {
	all := []Code{
		CodeNotFound, CodeNotRegularFile, CodeEmpty,
		CodeUnreadable, CodeNotWAV, CodeTruncated, CodeWrongExtension,
	}

	seen := make(map[Code]bool, len(all))
	for _, code := range all {
		if code == "" {
			t.Error("a code is empty")
		}
		if seen[code] {
			t.Errorf("code %q is duplicated", code)
		}
		seen[code] = true
	}
}

func TestCodeOfIgnoresUnrelatedErrors(t *testing.T) {
	if got := CodeOf(errors.New("something else")); got != "" {
		t.Errorf("CodeOf() = %q, want empty", got)
	}
	if got := CodeOf(nil); got != "" {
		t.Errorf("CodeOf(nil) = %q, want empty", got)
	}
}

func TestPrintableEscapesControlBytes(t *testing.T) {
	if got := printable([]byte{'I', 'D', '3', 0x04}); got != `ID3\x04` {
		t.Errorf("printable() = %q", got)
	}
}

func mustFail(t *testing.T, path string) error {
	t.Helper()

	_, err := Validate(path)
	if err == nil {
		t.Fatalf("Validate(%q) unexpectedly succeeded", path)
	}

	return err
}
