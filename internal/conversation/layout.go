package conversation

import (
	"path/filepath"
)

// File and directory names inside a conversation workspace. They are the on-disk
// contract from the project overview's Local Data Structure.
const (
	// MetadataFile holds the Conversation document.
	MetadataFile = "conversation.json"
	// OutlineFile is reserved for user notes. Created empty; no feature reads it
	// yet.
	OutlineFile = "outline.md"
	// ResourcesFile is reserved for linked resources. Created empty; no feature
	// reads it yet.
	ResourcesFile = "resources.json"

	// AudioDir holds the imported source and its optimized derivative.
	AudioDir = "audio"
	// JobsDir holds one document per transcription job.
	JobsDir = "jobs"
	// TranscriptDir holds the canonical transcript documents.
	TranscriptDir = "transcript"
	// ExportsDir holds generated exports such as Markdown.
	ExportsDir = "exports"
)

// Workspace is one conversation's directory and the paths within it.
type Workspace struct {
	// Dir is the conversation's own directory.
	Dir string
}

// NewWorkspace describes the workspace rooted at dir.
func NewWorkspace(dir string) Workspace {
	return Workspace{Dir: dir}
}

// MetadataPath is the conversation.json path.
func (w Workspace) MetadataPath() string { return filepath.Join(w.Dir, MetadataFile) }

// OutlinePath is the outline.md path.
func (w Workspace) OutlinePath() string { return filepath.Join(w.Dir, OutlineFile) }

// ResourcesPath is the resources.json path.
func (w Workspace) ResourcesPath() string { return filepath.Join(w.Dir, ResourcesFile) }

// AudioPath is the directory holding source and optimized audio.
func (w Workspace) AudioPath() string { return filepath.Join(w.Dir, AudioDir) }

// JobsPath is the directory holding job documents.
func (w Workspace) JobsPath() string { return filepath.Join(w.Dir, JobsDir) }

// TranscriptPath is the directory holding transcript documents.
func (w Workspace) TranscriptPath() string { return filepath.Join(w.Dir, TranscriptDir) }

// ExportsPath is the directory holding exports.
func (w Workspace) ExportsPath() string { return filepath.Join(w.Dir, ExportsDir) }

// Directories lists every directory a workspace must contain, in creation order.
func (w Workspace) Directories() []string {
	return []string{
		w.Dir,
		w.AudioPath(),
		w.JobsPath(),
		w.TranscriptPath(),
		w.ExportsPath(),
	}
}

// placeholders lists files created empty so a later version can populate them
// without a migration. Recording that they are reserved here keeps a reader from
// concluding the features exist.
func (w Workspace) placeholders() []string {
	return []string{
		w.OutlinePath(),
		w.ResourcesPath(),
	}
}
