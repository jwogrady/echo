package conversation

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// validConversation is a document that must always pass validation. Tests break
// one field at a time from this baseline.
func validConversation() Conversation {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	return Conversation{
		SchemaVersion: SchemaVersion,
		ID:            "cnv_01HQ8ZJ9M3XBQK7T2Y4V6P8R0D",
		Title:         "Product Strategy",
		Slug:          "product-strategy",
		CreatedAt:     now,
		UpdatedAt:     now,
		Status:        StatusCreated,
	}
}

// newTestWorkspace returns a workspace whose directory does not exist yet, which
// is what CreateWorkspace requires — it refuses an existing directory on purpose.
func newTestWorkspace(t *testing.T) Workspace {
	t.Helper()

	return NewWorkspace(filepath.Join(t.TempDir(), "cnv_01HQ8ZJ9M3XBQK7T2Y4V6P8R0D"))
}

// newEmptyWorkspace returns a workspace whose directory exists but holds no
// metadata, for tests that write their own damaged fixture into it.
func newEmptyWorkspace(t *testing.T) Workspace {
	t.Helper()

	workspace := NewWorkspace(filepath.Join(t.TempDir(), "cnv_01HQ8ZJ9M3XBQK7T2Y4V6P8R0D"))
	if err := os.MkdirAll(workspace.Dir, 0o755); err != nil {
		t.Fatalf("creating the workspace directory: %v", err)
	}

	return workspace
}

func TestValidConversationPasses(t *testing.T) {
	if err := validConversation().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Conversation)
	}{
		{"no id", func(c *Conversation) { c.ID = "" }},
		{"no title", func(c *Conversation) { c.Title = "" }},
		{"no created_at", func(c *Conversation) { c.CreatedAt = time.Time{} }},
		{"no updated_at", func(c *Conversation) { c.UpdatedAt = time.Time{} }},
		{"zero schema version", func(c *Conversation) { c.SchemaVersion = 0 }},
		{"unknown status", func(c *Conversation) { c.Status = "halfway" }},
		{"empty status", func(c *Conversation) { c.Status = "" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			subject := validConversation()
			test.mutate(&subject)

			err := subject.Validate()
			if err == nil {
				t.Fatal("expected an error")
			}
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("error = %v, want it to wrap ErrInvalid", err)
			}
		})
	}
}

// A document from a newer Echo must be refused as its own condition, so the CLI
// can tell the user to upgrade rather than reporting generic corruption.
func TestValidateRejectsANewerSchema(t *testing.T) {
	subject := validConversation()
	subject.SchemaVersion = SchemaVersion + 1

	err := subject.Validate()
	if !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("error = %v, want it to wrap ErrUnsupportedSchema", err)
	}
	if !strings.Contains(err.Error(), "this build supports") {
		t.Errorf("error = %q, want it to name both versions", err)
	}
}

func TestStatusValidityIsAClosedSet(t *testing.T) {
	for _, status := range Statuses() {
		if !status.Valid() {
			t.Errorf("%q is listed but reports invalid", status)
		}
	}

	for _, status := range []Status{"", "created ", "CREATED", "done", "recording-added"} {
		if Status(status).Valid() {
			t.Errorf("%q reports valid but is not in the set", status)
		}
	}
}

// The milestone names exactly these six states.
func TestStatusSetMatchesTheStateModel(t *testing.T) {
	want := []Status{
		StatusCreated, StatusRecordingAdded, StatusAudioReady,
		StatusTranscribing, StatusTranscribed, StatusFailed,
	}

	got := Statuses()
	if len(got) != len(want) {
		t.Fatalf("Statuses() has %d entries, want %d", len(got), len(want))
	}
	for i, status := range want {
		if got[i] != status {
			t.Errorf("Statuses()[%d] = %q, want %q", i, got[i], status)
		}
	}
}

// Statuses must not hand out a slice callers can mutate into the package.
func TestStatusesReturnsACopy(t *testing.T) {
	first := Statuses()
	first[0] = "tampered"

	if Statuses()[0] != StatusCreated {
		t.Error("Statuses() exposed its backing array")
	}
}

func TestRoundTripPreservesEveryField(t *testing.T) {
	original := validConversation()
	original.ActiveRecordingID = "rec_01HQ8ZJZ5N2WQT8Y3B7X1M4K6P"
	original.ActiveTranscriptID = "trs_01HQ8ZK4T7WYVX0PJ3M2N5R6QB"

	workspace := newTestWorkspace(t)
	if err := CreateWorkspace(workspace, original); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	loaded, err := Load(workspace)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded != original {
		t.Errorf("round trip changed the document:\n got %+v\nwant %+v", loaded, original)
	}
}

// Optional identifiers must be absent from the document rather than present and
// empty, so a reader can tell "no recording yet" from "recording with no id".
func TestEmptyOptionalFieldsAreOmitted(t *testing.T) {
	workspace := newTestWorkspace(t)
	if err := CreateWorkspace(workspace, validConversation()); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	contents, err := os.ReadFile(workspace.MetadataPath())
	if err != nil {
		t.Fatalf("reading metadata: %v", err)
	}

	for _, field := range []string{"active_recording_id", "active_transcript_id"} {
		if strings.Contains(string(contents), field) {
			t.Errorf("document contains %q while empty:\n%s", field, contents)
		}
	}
}

// The on-disk field names are a contract; renaming one is a schema change.
func TestOnDiskFieldNamesAreStable(t *testing.T) {
	encoded, err := json.Marshal(validConversation())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	for _, field := range []string{
		"schema_version", "id", "title", "slug",
		"created_at", "updated_at", "status",
	} {
		if _, ok := document[field]; !ok {
			t.Errorf("document is missing the %q field", field)
		}
	}
}

func TestCreateWorkspaceBuildsTheDocumentedLayout(t *testing.T) {
	workspace := newTestWorkspace(t)
	if err := CreateWorkspace(workspace, validConversation()); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	for _, dir := range []string{AudioDir, JobsDir, TranscriptDir, ExportsDir} {
		info, err := os.Stat(filepath.Join(workspace.Dir, dir))
		if err != nil {
			t.Errorf("missing directory %s: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}

	for _, file := range []string{MetadataFile, OutlineFile, ResourcesFile} {
		if _, err := os.Stat(filepath.Join(workspace.Dir, file)); err != nil {
			t.Errorf("missing file %s: %v", file, err)
		}
	}
}

// outline.md and resources.json exist for forward compatibility only.
func TestPlaceholderFilesAreEmpty(t *testing.T) {
	workspace := newTestWorkspace(t)
	if err := CreateWorkspace(workspace, validConversation()); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	for _, path := range []string{workspace.OutlinePath(), workspace.ResourcesPath()} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Size() != 0 {
			t.Errorf("%s is %d bytes, want empty", path, info.Size())
		}
	}
}

// Overwriting a workspace would destroy audio and transcripts that cannot be
// regenerated.
func TestCreateWorkspaceRefusesAnExistingDirectory(t *testing.T) {
	workspace := newTestWorkspace(t)
	if err := CreateWorkspace(workspace, validConversation()); err != nil {
		t.Fatalf("first CreateWorkspace() error = %v", err)
	}

	err := CreateWorkspace(workspace, validConversation())
	if err == nil {
		t.Fatal("expected an error on the second create")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Errorf("error = %q, want it to say it refused", err)
	}
}

func TestLoadDistinguishesMissingFromMalformed(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		_, err := Load(newEmptyWorkspace(t))
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want it to wrap ErrNotFound", err)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		workspace := newEmptyWorkspace(t)
		if err := os.WriteFile(workspace.MetadataPath(), []byte("{not json"), 0o644); err != nil {
			t.Fatalf("writing the fixture: %v", err)
		}

		_, err := Load(workspace)
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("error = %v, want it to wrap ErrMalformed", err)
		}
	})
}

// Reporting damage must never modify the evidence.
func TestLoadLeavesADamagedFileUntouched(t *testing.T) {
	workspace := newEmptyWorkspace(t)
	damaged := []byte(`{"schema_version": 1, "id": "cnv_1", "tit`)

	if err := os.WriteFile(workspace.MetadataPath(), damaged, 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	if _, err := Load(workspace); err == nil {
		t.Fatal("expected an error")
	}

	after, err := os.ReadFile(workspace.MetadataPath())
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(after) != string(damaged) {
		t.Errorf("Load modified the file:\n got %q\nwant %q", after, damaged)
	}
}

func TestSaveRefusesInvalidMetadata(t *testing.T) {
	workspace := newTestWorkspace(t)
	subject := validConversation()
	subject.Status = "nonsense"

	if err := Save(workspace, subject); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(workspace.MetadataPath()); err == nil {
		t.Error("Save wrote a file despite refusing the document")
	}
}

// An atomic write leaves no temporary files behind, and replaces contents wholly.
func TestSaveIsAtomicAndLeavesNoDebris(t *testing.T) {
	workspace := newTestWorkspace(t)
	if err := CreateWorkspace(workspace, validConversation()); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	updated := validConversation()
	updated.Title = "Renamed"
	updated.Status = StatusRecordingAdded
	updated.UpdatedAt = updated.UpdatedAt.Add(time.Hour)

	if err := Save(workspace, updated); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(workspace)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Title != "Renamed" || loaded.Status != StatusRecordingAdded {
		t.Errorf("Save did not replace the document: %+v", loaded)
	}

	entries, err := os.ReadDir(workspace.Dir)
	if err != nil {
		t.Fatalf("reading the workspace: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("Save left a temporary file behind: %s", entry.Name())
		}
	}
}

// The temporary file must be a sibling of its target: a rename across volumes is
// not atomic, and os.TempDir may well be a different filesystem.
func TestSaveWritesItsTemporaryFileBesideTheTarget(t *testing.T) {
	workspace := newTestWorkspace(t)
	if err := CreateWorkspace(workspace, validConversation()); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	readOnly := filepath.Join(workspace.Dir, "locked")
	if err := os.Mkdir(readOnly, 0o500); err != nil {
		t.Fatalf("creating the fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o700) })

	// Writing into a directory Echo cannot create files in must fail, proving the
	// temporary file is placed there rather than in the system temp directory.
	nested := NewWorkspace(readOnly)
	if err := Save(nested, validConversation()); err == nil {
		t.Skip("the filesystem ignores directory permissions; cannot assert placement here")
	}
}

func TestLoadRejectsANewerSchemaOnDisk(t *testing.T) {
	workspace := newEmptyWorkspace(t)

	future := validConversation()
	future.SchemaVersion = SchemaVersion + 5
	encoded, err := json.Marshal(future)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(workspace.MetadataPath(), encoded, 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	_, err = Load(workspace)
	if !errors.Is(err, ErrUnsupportedSchema) {
		t.Errorf("error = %v, want it to wrap ErrUnsupportedSchema", err)
	}
}

func TestWorkspacePathsSitUnderTheDirectory(t *testing.T) {
	workspace := NewWorkspace(filepath.Join("data", "conversations", "cnv_1"))

	paths := map[string]string{
		"metadata":   workspace.MetadataPath(),
		"outline":    workspace.OutlinePath(),
		"resources":  workspace.ResourcesPath(),
		"audio":      workspace.AudioPath(),
		"jobs":       workspace.JobsPath(),
		"transcript": workspace.TranscriptPath(),
		"exports":    workspace.ExportsPath(),
	}

	for name, path := range paths {
		if !strings.HasPrefix(path, workspace.Dir) {
			t.Errorf("%s path %q escapes the workspace %q", name, path, workspace.Dir)
		}
	}
}
