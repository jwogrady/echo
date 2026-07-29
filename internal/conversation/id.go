package conversation

import (
	"crypto/rand"
	"fmt"
	"strings"
	"unicode"
)

// IDPrefix marks a conversation identifier. Later milestones add rec_ and trs_
// for recordings and transcripts, so a bare identifier is never ambiguous about
// what it names.
const IDPrefix = "cnv_"

// idAlphabet is Crockford base32: no I, L, O, or U, so an identifier cannot be
// misread or mistyped into a different valid one when a user copies it from a
// terminal to select a conversation by prefix.
const idAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// idBodyLength is the number of random characters after the prefix. At 26
// characters over a 32-symbol alphabet that is 130 bits, so collisions are not a
// practical concern even though IDs are generated independently on one machine.
const idBodyLength = 26

// NewID returns a fresh conversation identifier.
//
// Randomness comes from crypto/rand rather than a timestamp or a counter: IDs
// double as directory names and as the thing a user types a prefix of, so two
// conversations created in the same second must not share a prefix.
func NewID() (string, error) {
	body, err := randomString(idBodyLength)
	if err != nil {
		return "", fmt.Errorf("generating a conversation id: %w", err)
	}

	return IDPrefix + body, nil
}

// randomString draws length characters from idAlphabet without modulo bias.
func randomString(length int) (string, error) {
	// 32 divides 256 exactly, so masking the low 5 bits is uniform and no
	// rejection sampling is needed.
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.Grow(length)
	for _, b := range buffer {
		builder.WriteByte(idAlphabet[b&0x1f])
	}

	return builder.String(), nil
}

// ValidID reports whether id has the shape NewID produces. It is a guard against
// path traversal as much as a format check: identifiers become directory names,
// so anything containing a separator or a dot segment must be refused.
func ValidID(id string) bool {
	body, ok := strings.CutPrefix(id, IDPrefix)
	if !ok || len(body) != idBodyLength {
		return false
	}

	for _, r := range body {
		if !strings.ContainsRune(idAlphabet, r) {
			return false
		}
	}

	return true
}

// slugMaxLength bounds a slug so a title cannot produce an unwieldy path. Windows
// has a total path limit and a conversation directory sits several levels deep.
const slugMaxLength = 60

// Slugify renders a title as a filesystem- and URL-safe string.
//
// The slug is for human readability only — it is never an identifier, is never
// used to look a conversation up, and two conversations may share one. A title
// that reduces to nothing yields "untitled" rather than an empty string.
func Slugify(title string) string {
	var builder strings.Builder
	builder.Grow(len(title))

	lastWasHyphen := false
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastWasHyphen = false
		case !lastWasHyphen && builder.Len() > 0:
			builder.WriteByte('-')
			lastWasHyphen = true
		}
	}

	slug := strings.Trim(builder.String(), "-")

	if len(slug) > slugMaxLength {
		slug = strings.Trim(slug[:slugMaxLength], "-")
	}

	if slug == "" {
		return "untitled"
	}

	return slug
}
