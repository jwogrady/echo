package audio

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwogrady/echo/internal/conversation"
)

// newTestWorkspace builds a conversation workspace on disk to import into.
func newTestWorkspace(t *testing.T) conversation.Workspace {
	t.Helper()

	workspace := conversation.NewWorkspace(filepath.Join(t.TempDir(), "cnv_TEST"))
	now := time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC)

	err := conversation.CreateWorkspace(workspace, conversation.Conversation{
		SchemaVersion: conversation.SchemaVersion,
		ID:            "cnv_AAAAAAAAAAAAAAAAAAAAAAAAAA",
		Title:         "Test",
		Slug:          "test",
		CreatedAt:     now,
		UpdatedAt:     now,
		Status:        conversation.StatusCreated,
	})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	return workspace
}

// testImporter uses a fixed clock and a deterministic id.
func testImporter() *Importer {
	moment := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)

	return &Importer{
		Now: func() time.Time { moment = moment.Add(time.Minute); return moment },
		NewID: func() (string, error) {
			return RecordingIDPrefix + strings.Repeat("A", 26), nil
		},
	}
}

// validSource writes a WAV fixture and validates it.
func validSource(t *testing.T, name string, payload []byte) Source {
	t.Helper()

	path := writeFixture(t, name, wavBytes(payload))
	source, err := Validate(path)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	return source
}

func TestImportCopiesAndVerifies(t *testing.T) {
	workspace := newTestWorkspace(t)
	source := validSource(t, "recording.wav", silence())

	recording, err := testImporter().Import(workspace, "cnv_AAAAAAAAAAAAAAAAAAAAAAAAAA", source, false)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if !ValidRecordingID(recording.ID) {
		t.Errorf("ID = %q, not a valid recording id", recording.ID)
	}
	if recording.OriginalFilename != "recording.wav" {
		t.Errorf("OriginalFilename = %q", recording.OriginalFilename)
	}
	if len(recording.SHA256) != 64 {
		t.Errorf("SHA256 = %q, want 64 hex characters", recording.SHA256)
	}
	if recording.SourceProperties != nil {
		t.Error("SourceProperties should be nil until inspected, not a zero value")
	}

	// The copy must be byte-identical to the original.
	original, err := os.ReadFile(source.Path)
	if err != nil {
		t.Fatalf("reading the original: %v", err)
	}
	copied, err := os.ReadFile(SourcePath(workspace))
	if err != nil {
		t.Fatalf("reading the copy: %v", err)
	}
	if string(original) != string(copied) {
		t.Error("the imported copy differs from the source")
	}
	if recording.SizeBytes != int64(len(original)) {
		t.Errorf("SizeBytes = %d, want %d", recording.SizeBytes, len(original))
	}

	loaded, err := LoadRecording(workspace)
	if err != nil {
		t.Fatalf("LoadRecording() error = %v", err)
	}
	if loaded.ID != recording.ID {
		t.Errorf("round trip changed the id: %q vs %q", loaded.ID, recording.ID)
	}
}

// The user's file is theirs. Echo reads it and nothing else.
func TestImportNeverTouchesTheSource(t *testing.T) {
	workspace := newTestWorkspace(t)
	source := validSource(t, "recording.wav", silence())

	before, err := os.Stat(source.Path)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}
	contentsBefore, err := os.ReadFile(source.Path)
	if err != nil {
		t.Fatalf("reading before: %v", err)
	}

	if _, err := testImporter().Import(workspace, "cnv_A", source, false); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	after, err := os.Stat(source.Path)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	contentsAfter, err := os.ReadFile(source.Path)
	if err != nil {
		t.Fatalf("reading after: %v", err)
	}

	if !before.ModTime().Equal(after.ModTime()) {
		t.Error("import changed the source's modification time")
	}
	if string(contentsBefore) != string(contentsAfter) {
		t.Error("import changed the source's contents")
	}
}

func TestReimportingTheSameFileIsANoOp(t *testing.T) {
	workspace := newTestWorkspace(t)
	source := validSource(t, "recording.wav", silence())
	importer := testImporter()

	first, err := importer.Import(workspace, "cnv_A", source, false)
	if err != nil {
		t.Fatalf("first Import() error = %v", err)
	}

	again, err := importer.Import(workspace, "cnv_A", source, false)
	if !errors.Is(err, ErrAlreadyImported) {
		t.Fatalf("error = %v, want ErrAlreadyImported", err)
	}
	if again.ID != first.ID {
		t.Errorf("the no-op returned a different recording: %q vs %q", again.ID, first.ID)
	}
}

// Silently discarding audio the user may not have elsewhere is unacceptable.
func TestADifferentFileIsRefusedWithoutReplace(t *testing.T) {
	workspace := newTestWorkspace(t)
	importer := testImporter()

	first := validSource(t, "first.wav", silence())
	if _, err := importer.Import(workspace, "cnv_A", first, false); err != nil {
		t.Fatalf("first Import() error = %v", err)
	}

	second := validSource(t, "second.wav", make([]byte, 2048))
	_, err := importer.Import(workspace, "cnv_A", second, false)
	if !errors.Is(err, ErrDifferentRecording) {
		t.Fatalf("error = %v, want ErrDifferentRecording", err)
	}
	if !strings.Contains(err.Error(), "--replace") {
		t.Errorf("error = %q, want it to name the flag that would allow this", err)
	}

	// The original recording must survive the refusal.
	loaded, err := LoadRecording(workspace)
	if err != nil {
		t.Fatalf("LoadRecording() error = %v", err)
	}
	if loaded.OriginalFilename != "first.wav" {
		t.Errorf("OriginalFilename = %q, want the original to survive", loaded.OriginalFilename)
	}
}

func TestReplaceOverwritesDeliberately(t *testing.T) {
	workspace := newTestWorkspace(t)
	importer := testImporter()

	first := validSource(t, "first.wav", silence())
	if _, err := importer.Import(workspace, "cnv_A", first, false); err != nil {
		t.Fatalf("first Import() error = %v", err)
	}

	second := validSource(t, "second.wav", make([]byte, 2048))
	replaced, err := importer.Import(workspace, "cnv_A", second, true)
	if err != nil {
		t.Fatalf("Import(replace) error = %v", err)
	}
	if replaced.OriginalFilename != "second.wav" {
		t.Errorf("OriginalFilename = %q, want the replacement", replaced.OriginalFilename)
	}

	copied, err := os.ReadFile(SourcePath(workspace))
	if err != nil {
		t.Fatalf("reading the copy: %v", err)
	}
	if int64(len(copied)) != replaced.SizeBytes {
		t.Error("source.wav was not replaced with the new audio")
	}
}

// A checksum mismatch means the copy cannot be trusted, so it must not survive.
func TestChecksumMismatchRemovesTheCopy(t *testing.T) {
	workspace := newTestWorkspace(t)
	source := validSource(t, "recording.wav", silence())

	importer := testImporter()
	// Corrupt the copy between writing and verifying, simulating a bad disk.
	importer.Progress = func(stage Stage) {
		if stage == StageVerifying {
			_ = os.WriteFile(SourcePath(workspace), []byte("corrupted"), 0o644)
		}
	}

	_, err := importer.Import(workspace, "cnv_A", source, false)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("error = %v, want ErrChecksumMismatch", err)
	}

	if _, err := os.Stat(SourcePath(workspace)); !errors.Is(err, os.ErrNotExist) {
		t.Error("the unverifiable copy was left in place")
	}
	if _, err := LoadRecording(workspace); !errors.Is(err, ErrNoRecording) {
		t.Error("a recording document was written for an unverifiable copy")
	}
}

// A failure names the stage, because "import failed" is not actionable.
func TestFailuresNameTheirStage(t *testing.T) {
	workspace := newTestWorkspace(t)
	source := validSource(t, "recording.wav", silence())

	// Removing the source between validation and hashing makes hashing fail.
	if err := os.Remove(source.Path); err != nil {
		t.Fatalf("removing the source: %v", err)
	}

	_, err := testImporter().Import(workspace, "cnv_A", source, false)
	if err == nil {
		t.Fatal("expected an error")
	}

	var staged *StageError
	if !errors.As(err, &staged) {
		t.Fatalf("error = %v, want a StageError", err)
	}
	if staged.Stage != StageHashing {
		t.Errorf("Stage = %q, want %q", staged.Stage, StageHashing)
	}
	if !strings.Contains(err.Error(), string(StageHashing)) {
		t.Errorf("message = %q, want it to name the stage", err)
	}
}

// A failed import must leave nothing that status would report as a recording.
func TestFailedImportLeavesNoPartialState(t *testing.T) {
	workspace := newTestWorkspace(t)
	source := validSource(t, "recording.wav", silence())

	if err := os.Remove(source.Path); err != nil {
		t.Fatalf("removing the source: %v", err)
	}

	if _, err := testImporter().Import(workspace, "cnv_A", source, false); err == nil {
		t.Fatal("expected an error")
	}

	entries, err := os.ReadDir(workspace.AudioPath())
	if err != nil {
		t.Fatalf("reading the audio directory: %v", err)
	}
	for _, entry := range entries {
		t.Errorf("audio/ should be empty after a failed import, found %s", entry.Name())
	}
}

// Treating a damaged document as "no recording" would overwrite the audio.
func TestDamagedRecordingDocumentBlocksImport(t *testing.T) {
	workspace := newTestWorkspace(t)
	source := validSource(t, "recording.wav", silence())

	if err := os.WriteFile(recordingPath(workspace), []byte("{broken"), 0o644); err != nil {
		t.Fatalf("writing the damaged fixture: %v", err)
	}

	_, err := testImporter().Import(workspace, "cnv_A", source, false)
	if !errors.Is(err, ErrRecordingMalformed) {
		t.Errorf("error = %v, want ErrRecordingMalformed", err)
	}
	if _, err := os.Stat(SourcePath(workspace)); !errors.Is(err, os.ErrNotExist) {
		t.Error("import proceeded despite an unreadable recording document")
	}
}

func TestRecordingValidationRejectsImpossibleDocuments(t *testing.T) {
	base := Recording{
		SchemaVersion:    RecordingSchemaVersion,
		ID:               RecordingIDPrefix + strings.Repeat("A", 26),
		OriginalFilename: "recording.wav",
		SHA256:           strings.Repeat("a", 64),
		SizeBytes:        100,
		ImportedAt:       time.Now().UTC(),
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("the baseline should be valid: %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*Recording)
		wantErr error
	}{
		{"newer schema", func(r *Recording) { r.SchemaVersion = 99 }, ErrRecordingUnsupportedSchema},
		{"zero schema", func(r *Recording) { r.SchemaVersion = 0 }, ErrRecordingInvalid},
		{"bad id", func(r *Recording) { r.ID = "nope" }, ErrRecordingInvalid},
		{"conversation id as recording id", func(r *Recording) { r.ID = "cnv_" + strings.Repeat("A", 26) }, ErrRecordingInvalid},
		{"no filename", func(r *Recording) { r.OriginalFilename = "" }, ErrRecordingInvalid},
		{"short checksum", func(r *Recording) { r.SHA256 = "abc" }, ErrRecordingInvalid},
		{"zero size", func(r *Recording) { r.SizeBytes = 0 }, ErrRecordingInvalid},
		{"no timestamp", func(r *Recording) { r.ImportedAt = time.Time{} }, ErrRecordingInvalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			subject := base
			test.mutate(&subject)

			if err := subject.Validate(); !errors.Is(err, test.wantErr) {
				t.Errorf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

// Unset audio properties must be absent, not zero: a 0 Hz sample rate would look
// like a real measurement.
func TestUnprobedPropertiesAreOmittedFromDisk(t *testing.T) {
	workspace := newTestWorkspace(t)
	source := validSource(t, "recording.wav", silence())

	if _, err := testImporter().Import(workspace, "cnv_A", source, false); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	contents, err := os.ReadFile(recordingPath(workspace))
	if err != nil {
		t.Fatalf("reading the document: %v", err)
	}

	for _, field := range []string{"source_properties", "optimized_properties", "optimized_sha256"} {
		if strings.Contains(string(contents), field) {
			t.Errorf("document contains %q before it is known:\n%s", field, contents)
		}
	}
}

func TestRecordingIDsAreDistinctFromConversationIDs(t *testing.T) {
	id, err := NewRecordingID()
	if err != nil {
		t.Fatalf("NewRecordingID() error = %v", err)
	}

	if !ValidRecordingID(id) {
		t.Errorf("%q is not a valid recording id", id)
	}
	if conversation.ValidID(id) {
		t.Errorf("%q also validates as a conversation id", id)
	}
}

func TestRemoveRecordingIsIdempotent(t *testing.T) {
	workspace := newTestWorkspace(t)

	if err := RemoveRecording(workspace); err != nil {
		t.Errorf("removing nothing should succeed, got %v", err)
	}

	source := validSource(t, "recording.wav", silence())
	if _, err := testImporter().Import(workspace, "cnv_A", source, false); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if err := RemoveRecording(workspace); err != nil {
		t.Fatalf("RemoveRecording() error = %v", err)
	}
	if _, err := LoadRecording(workspace); !errors.Is(err, ErrNoRecording) {
		t.Errorf("error = %v, want ErrNoRecording", err)
	}
}
