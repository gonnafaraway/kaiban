package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMergeConfluenceSection_appendAndReplace(t *testing.T) {
	got := MergeConfluenceSection("", "kaiban:1", "<h2>A</h2>")
	if !strings.Contains(got, "<!-- kaiban:1 -->") || !strings.Contains(got, "<h2>A</h2>") {
		t.Fatalf("append empty: %s", got)
	}
	got = MergeConfluenceSection(got+"keep", "kaiban:1", "<h2>B</h2>")
	if strings.Contains(got, "<h2>A</h2>") || !strings.Contains(got, "<h2>B</h2>") || !strings.Contains(got, "keep") {
		t.Fatalf("replace: %s", got)
	}
}

func TestAppendConfluencePage_put(t *testing.T) {
	var putBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "2193774518", "type": "page", "title": "BT",
				"space":   map[string]any{"key": "VKCSP"},
				"version": map[string]any{"number": 1},
				"body":    map[string]any{"storage": map[string]any{"value": ""}},
			})
		case http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			putBody = string(b)
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"2193774518"}`))
		default:
			w.WriteHeader(405)
		}
	}))
	defer srv.Close()
	if _, err := AppendConfluencePage(t.Context(), srv.URL, "-", "tok", "2193774518", "kaiban:t:c", "Kaiban / Dev", "hello report"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(putBody, "hello report") || !strings.Contains(putBody, `"number":2`) {
		t.Fatalf("put %s", putBody)
	}
}
