package integration_test

import (
	"testing"

	"kaiban/internal/api/integration"
)

func TestExtractRemoteURL(t *testing.T) {
	gh := `status=201 body={"html_url":"https://github.com/acme/app/pull/1","number":1}`
	if got := integration.ExtractRemoteURL(gh); got != "https://github.com/acme/app/pull/1" {
		t.Fatalf("github: %q", got)
	}
	gl := `status=201 body={"web_url":"https://gitlab.com/acme/app/-/merge_requests/2"}`
	if got := integration.ExtractRemoteURL(gl); got != "https://gitlab.com/acme/app/-/merge_requests/2" {
		t.Fatalf("gitlab: %q", got)
	}
	if got := integration.ExtractRemoteURL("status=400 body={}"); got != "" {
		t.Fatalf("empty: %q", got)
	}
}
