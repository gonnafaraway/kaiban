package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPToolCall_errorOnHTTPStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"token expired"}`))
	}))
	defer srv.Close()

	tool := HTTPTool{name: "jira_search", method: http.MethodGet, url: srv.URL}
	out, err := tool.Call(t.Context(), `{"jql":"project = X"}`)
	if err == nil {
		t.Fatalf("expected error, got %q", out)
	}
	if out != "" {
		t.Fatalf("expected empty output, got %q", out)
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "token expired") {
		t.Fatalf("error lacks status or body: %v", err)
	}
}

func TestHTTPToolCall_success(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("jql")
		_, _ = w.Write([]byte(`{"issues":[]}`))
	}))
	defer srv.Close()

	tool := HTTPTool{name: "jira_search", method: http.MethodGet, url: srv.URL}
	out, err := tool.Call(t.Context(), `{"jql":"project = X"}`)
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery != "project = X" {
		t.Fatalf("query %q", gotQuery)
	}
	if !strings.Contains(out, "status=200") || !strings.Contains(out, `"issues"`) {
		t.Fatalf("out %q", out)
	}
}

func TestHealthProbeURL(t *testing.T) {
	cases := map[string]string{
		KindJira:       "https://acme.atlassian.net/rest/api/2/myself",
		KindConfluence: "https://acme.atlassian.net/rest/api/user/current",
	}
	for kind, want := range cases {
		if got := HealthProbeURL(kind, "https://acme.atlassian.net/"); got != want {
			t.Fatalf("%s: got %q want %q", kind, got, want)
		}
	}
	if got := HealthProbeURL(KindGitLab, "https://gitlab.com"); got != "https://gitlab.com/api/v4/user" {
		t.Fatalf("gitlab: %q", got)
	}
	if got := HealthProbeURL(KindGitHub, "https://api.github.com"); got != "https://api.github.com/user" {
		t.Fatalf("github: %q", got)
	}
	if got := HealthProbeURL(KindGitHub, "https://github.acme.com"); got != "https://github.acme.com/api/v3/user" {
		t.Fatalf("github enterprise: %q", got)
	}
	if got := HealthProbeURL(KindJira, ""); got != "" {
		t.Fatalf("empty base: %q", got)
	}
}
