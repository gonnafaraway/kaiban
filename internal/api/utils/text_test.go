package textutil

import "testing"

func TestTruncateRunes(t *testing.T) {
	got := TruncateRunes("абвгде", 3)
	if got != "абв..." {
		t.Fatalf("got %q", got)
	}
	if TruncateRunes("short", 10) != "short" {
		t.Fatal("short unchanged")
	}
	if TruncateRunes("abc", 0) != "" {
		t.Fatal("non-positive limit yields empty string")
	}
}
