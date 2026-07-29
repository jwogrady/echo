package conversation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrNotFound means no conversation metadata exists at the expected path.
var ErrNotFound = errors.New("conversation not found")

// ErrMalformed means the metadata file exists but is not usable JSON. It is
// distinct from ErrInvalid, which means the JSON parsed but said something wrong.
var ErrMalformed = errors.New("conversation metadata is not valid JSON")

// Load reads and validates the conversation metadata in the workspace.
//
// Every failure leaves the file exactly as it was. A conversation directory can
// hold the user's only record of their thinking, so a document Echo cannot
// understand is reported, never repaired.
func Load(workspace Workspace) (Conversation, error) {
	path := workspace.MetadataPath()

	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Conversation{}, fmt.Errorf("%w: no %s in %s", ErrNotFound, MetadataFile, workspace.Dir)
		}

		return Conversation{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var loaded Conversation
	if err := json.Unmarshal(contents, &loaded); err != nil {
		return Conversation{}, fmt.Errorf("%w: %s: %w", ErrMalformed, path, err)
	}

	if err := loaded.Validate(); err != nil {
		return Conversation{}, fmt.Errorf("%s: %w", path, err)
	}

	return loaded, nil
}

// Save writes the conversation metadata atomically.
//
// The document is written to a temporary file in the destination directory and
// then renamed over the target. Same-directory keeps the rename on one volume,
// where it is atomic; rename-over is the operation that replaces a file correctly
// on Windows. A crash therefore leaves either the previous document or the new
// one, never a truncated file.
func Save(workspace Workspace, c Conversation) error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("refusing to write invalid metadata: %w", err)
	}

	encoded, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding conversation %s: %w", c.ID, err)
	}
	encoded = append(encoded, '\n')

	return WriteFileAtomic(workspace.MetadataPath(), encoded)
}

// WriteFileAtomic replaces path's contents in a single filesystem operation.
//
// Exported so every part of Echo that persists state uses this one discipline
// rather than reimplementing it: the temporary file is created beside the target
// (a cross-volume rename is not atomic) and fsynced before the rename (without
// which the rename can be durable while the contents are not).
func WriteFileAtomic(path string, contents []byte) error {
	dir := filepath.Dir(path)

	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating a temporary file in %s: %w", dir, err)
	}
	tempName := temp.Name()

	// Any failure from here on must not leave the temporary file behind.
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}

	if _, err := temp.Write(contents); err != nil {
		cleanup()
		return fmt.Errorf("writing %s: %w", tempName, err)
	}

	// fsync before the rename: without it the rename can be durable while the
	// contents are not, which would survive a crash as an empty file.
	if err := temp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("flushing %s: %w", tempName, err)
	}

	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("closing %s: %w", tempName, err)
	}

	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("replacing %s: %w", path, err)
	}

	return nil
}

// CreateWorkspace builds the directory tree and placeholder files for a new
// conversation, then writes its metadata.
//
// It refuses to touch an existing workspace: overwriting one would destroy
// recordings and transcripts that cannot be regenerated.
func CreateWorkspace(workspace Workspace, c Conversation) error {
	if _, err := os.Stat(workspace.Dir); err == nil {
		return fmt.Errorf("%s already exists; refusing to overwrite it", workspace.Dir)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("checking %s: %w", workspace.Dir, err)
	}

	for _, dir := range workspace.Directories() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	for _, placeholder := range workspace.placeholders() {
		if err := createEmpty(placeholder); err != nil {
			return err
		}
	}

	return Save(workspace, c)
}

// createEmpty makes an empty file without clobbering an existing one.
func createEmpty(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}

		return fmt.Errorf("creating %s: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}

	return nil
}
