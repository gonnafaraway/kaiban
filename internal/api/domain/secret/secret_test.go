package secret

import "testing"

func TestMask(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"abc":              "****",
		"abcd":             "****",
		"sk-live-12345678": "****5678",
	}
	for in, want := range cases {
		if got := Mask(in); got != want {
			t.Fatalf("Mask(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsMasked(t *testing.T) {
	if !IsMasked(Mask("sk-live-12345678")) {
		t.Fatal("mask of a secret must read back as masked")
	}
	if IsMasked("sk-live-12345678") {
		t.Fatal("a real secret must not read as masked")
	}
}
