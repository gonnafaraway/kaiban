package kanban

import "testing"

func TestScrubSecretMap(t *testing.T) {
	in := map[string]string{
		"token":  "secret-value",
		"masked": "****abcd",
		"empty":  "",
	}
	out := scrubSecretMap(in)
	if out["token"] != "secret-value" {
		t.Fatalf("token=%q", out["token"])
	}
	if _, ok := out["masked"]; ok {
		t.Fatal("masked secret should be dropped")
	}
	if out["empty"] != "" {
		t.Fatalf("empty=%q", out["empty"])
	}
}
