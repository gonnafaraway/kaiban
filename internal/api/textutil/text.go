// Package textutil holds the small string helpers shared across layers.
package textutil

// TruncateRunes cuts s to n runes and marks the cut with an ellipsis. It counts
// runes, not bytes, so Cyrillic text is not chopped mid-character.
func TruncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
