package column

import "testing"

func TestValidateOutputs(t *testing.T) {
	fields := []OutputField{
		{Key: "summary", Label: "Summary", Required: true, Type: OutputString},
		{Key: "link", Label: "Link", Required: false, Type: OutputURL},
	}
	if err := ValidateOutputs(fields, map[string]string{"summary": "ok"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOutputs(fields, map[string]string{}); err == nil {
		t.Fatal("expected missing required")
	}
	if err := ValidateOutputs(fields, map[string]string{"summary": "x", "link": "not-a-url"}); err == nil {
		t.Fatal("expected bad url")
	}
	if err := ValidateOutputs(fields, map[string]string{"summary": "x", "link": "https://example.com/a"}); err != nil {
		t.Fatal(err)
	}
}
