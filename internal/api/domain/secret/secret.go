// Package secret hides stored credentials behind a stable placeholder so the
// API can echo a value back without leaking it.
package secret

import "strings"

const prefix = "****"

// Mask returns the placeholder shown instead of a stored secret: the last four
// characters with everything before them replaced by asterisks.
func Mask(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return prefix
	}
	return prefix + v[len(v)-4:]
}

// IsMasked reports whether v is a placeholder produced by Mask — the client
// echoed back what it was shown instead of sending a new secret.
func IsMasked(v string) bool {
	return strings.HasPrefix(v, prefix)
}
