package integration_test

import (
	"testing"

	"kaiban/internal/api/integration"
)

func TestParseJiraIssue(t *testing.T) {
	cases := map[string]string{
		"VKTCORE-12260": "VKTCORE-12260",
		"https://jira.vk.team/browse/VKTCORE-12260":                    "VKTCORE-12260",
		"https://jira.vk.team/browse/VKTCORE-12260?focusedCommentId=1": "VKTCORE-12260",
	}
	for in, want := range cases {
		got, err := integration.ParseJiraIssue(in)
		if err != nil || got != want {
			t.Fatalf("%s: got %q %v want %q", in, got, err, want)
		}
	}
	if _, err := integration.ParseJiraIssue(""); err == nil {
		t.Fatal("empty must fail")
	}
}

func TestParseConfluencePageID(t *testing.T) {
	got, err := integration.ParseConfluencePageID("https://confluence.vk.team/pages/viewpage.action?pageId=2193774518")
	if err != nil || got != "2193774518" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = integration.ParseConfluencePageID("2193774518")
	if err != nil || got != "2193774518" {
		t.Fatalf("id: %q %v", got, err)
	}
}

func TestParseGitLabProject(t *testing.T) {
	got, err := integration.ParseGitLabProject("https://gitlab.corp.mail.ru/a.shorokhov/test-project")
	if err != nil || got != "a.shorokhov/test-project" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = integration.ParseGitLabProject("a.shorokhov/test-project")
	if err != nil || got != "a.shorokhov/test-project" {
		t.Fatalf("path: %q %v", got, err)
	}
	got, err = integration.ParseGitLabProject("https://gitlab.corp.mail.ru/a.shorokhov/test-project.git")
	if err != nil || got != "a.shorokhov/test-project" {
		t.Fatalf("git suffix: %q %v", got, err)
	}
}

func TestParseGitHubRepo(t *testing.T) {
	cases := map[string]string{
		"https://github.com/acme/app":     "acme/app",
		"https://github.com/acme/app.git": "acme/app",
		"acme/app":                        "acme/app",
		"git@github.com:acme/app.git":     "acme/app",
	}
	for in, want := range cases {
		got, err := integration.ParseGitHubRepo(in)
		if err != nil || got != want {
			t.Fatalf("%s: got %q %v want %q", in, got, err, want)
		}
	}
	if _, err := integration.ParseGitHubRepo(""); err == nil {
		t.Fatal("empty must fail")
	}
}

func TestIsGitHubHost(t *testing.T) {
	if !integration.IsGitHubHost("https://github.com/acme/app") {
		t.Fatal("github.com")
	}
	if integration.IsGitHubHost("https://gitlab.com/acme/app") {
		t.Fatal("gitlab must be false")
	}
}
