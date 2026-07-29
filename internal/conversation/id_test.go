package conversation

import (
	"strings"
	"testing"
)

func TestNewIDHasTheExpectedShape(t *testing.T) {
	id, err := NewID()
	if err != nil {
		t.Fatalf("NewID() error = %v", err)
	}

	if !strings.HasPrefix(id, IDPrefix) {
		t.Errorf("id = %q, want the %q prefix", id, IDPrefix)
	}
	if !ValidID(id) {
		t.Errorf("NewID() produced %q, which ValidID rejects", id)
	}
}

// IDs double as directory names and as the thing a user types a prefix of, so
// two created in the same instant must not collide or share a prefix.
func TestNewIDsAreDistinct(t *testing.T) {
	const count = 2000

	seen := make(map[string]bool, count)
	prefixes := make(map[string]bool, count)

	for range count {
		id, err := NewID()
		if err != nil {
			t.Fatalf("NewID() error = %v", err)
		}
		if seen[id] {
			t.Fatalf("NewID() repeated %q", id)
		}
		seen[id] = true

		// A 6-character prefix must stay usable for selection.
		prefix := id[:len(IDPrefix)+6]
		if prefixes[prefix] {
			t.Logf("6-character prefix %q recurred within %d ids", prefix, count)
		}
		prefixes[prefix] = true
	}
}

// The alphabet excludes I, L, O and U so an id cannot be misread into a
// different valid one when copied out of a terminal.
func TestIDAlphabetExcludesAmbiguousLetters(t *testing.T) {
	for _, excluded := range []rune{'I', 'L', 'O', 'U'} {
		if strings.ContainsRune(idAlphabet, excluded) {
			t.Errorf("alphabet contains the ambiguous letter %q", excluded)
		}
	}

	if len(idAlphabet) != 32 {
		t.Errorf("alphabet is %d symbols; masking 5 bits is only unbiased at 32", len(idAlphabet))
	}
}

func TestValidIDRejectsBadInput(t *testing.T) {
	valid := IDPrefix + strings.Repeat("A", idBodyLength)

	if !ValidID(valid) {
		t.Fatalf("ValidID(%q) = false, want true", valid)
	}

	for _, candidate := range []string{
		"",
		"cnv_",
		"cnv_TOOSHORT",
		valid + "X",
		strings.Repeat("A", idBodyLength),
		"rec_" + strings.Repeat("A", idBodyLength),
		// Lowercase is not in the alphabet.
		IDPrefix + strings.Repeat("a", idBodyLength),
		// Excluded letters must not pass.
		IDPrefix + "I" + strings.Repeat("A", idBodyLength-1),
		// Separators and dot segments must never reach the filesystem.
		"cnv_" + strings.Repeat("A", idBodyLength-2) + "/..",
		"../" + valid,
	} {
		if ValidID(candidate) {
			t.Errorf("ValidID(%q) = true, want false", candidate)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Product Strategy", "product-strategy"},
		{"Q3 planning: notes & ideas!", "q3-planning-notes-ideas"},
		{"   leading and trailing   ", "leading-and-trailing"},
		{"multiple---separators", "multiple-separators"},
		{"UPPERCASE", "uppercase"},
		{"with123numbers", "with123numbers"},
		{"!!!", "untitled"},
		{"", "untitled"},
		{"   ", "untitled"},
		{"café résumé", "café-résumé"},
		{"emoji 🎙 recording", "emoji-recording"},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			if got := Slugify(test.title); got != test.want {
				t.Errorf("Slugify(%q) = %q, want %q", test.title, got, test.want)
			}
		})
	}
}

// A long title must not produce an unwieldy path; Windows limits total length
// and a conversation directory sits several levels down.
func TestSlugifyBoundsLength(t *testing.T) {
	slug := Slugify(strings.Repeat("long title ", 40))

	if len(slug) > slugMaxLength {
		t.Errorf("slug is %d characters, want at most %d", len(slug), slugMaxLength)
	}
	if strings.HasSuffix(slug, "-") {
		t.Errorf("slug %q ends with a separator", slug)
	}
}

// The slug is decoration. Nothing may depend on it being unique.
func TestSlugifyIsNotAnIdentifier(t *testing.T) {
	if Slugify("Product Strategy") != Slugify("product strategy") {
		t.Error("expected these titles to share a slug")
	}
}
