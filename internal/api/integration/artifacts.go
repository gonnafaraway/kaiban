package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func PostJiraComment(ctx context.Context, base, email, token, issue, text string) (string, error) {
	key, err := ParseJiraIssue(issue)
	if err != nil {
		return "", err
	}
	u := strings.TrimRight(base, "/") + "/rest/api/2/issue/" + url.QueryEscape(key) + "/comment"
	payload, _ := json.Marshal(map[string]any{"body": text})
	return doJSON(ctx, http.MethodPost, u, AtlassianAuth(email, token), payload)
}

func FetchConfluencePage(ctx context.Context, base, email, token, pageRef string) (string, error) {
	id, err := ParseConfluencePageID(pageRef)
	if err != nil {
		return "", err
	}
	u := strings.TrimRight(base, "/") + "/rest/api/content/" + url.QueryEscape(id) + "?expand=body.storage,version,title"
	raw, err := doJSON(ctx, http.MethodGet, u, AtlassianAuth(email, token), nil)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Title string `json:"title"`
		Body  struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}
	_ = json.Unmarshal([]byte(stripStatus(raw)), &parsed)
	text := parsed.Title + "\n" + parsed.Body.Storage.Value
	if len(text) > 12000 {
		text = text[:12000] + "\n…"
	}
	return text, nil
}

func PostConfluenceComment(ctx context.Context, base, email, token, pageRef, htmlBody string) (string, error) {
	id, err := ParseConfluencePageID(pageRef)
	if err != nil {
		return "", err
	}
	u := strings.TrimRight(base, "/") + "/rest/api/content"
	payload, _ := json.Marshal(map[string]any{
		"type": "comment",
		"container": map[string]any{
			"type": "page",
			"id":   id,
		},
		"body": map[string]any{
			"storage": map[string]any{
				"value":          htmlBody,
				"representation": "storage",
			},
		},
	})
	return doJSON(ctx, http.MethodPost, u, AtlassianAuth(email, token), payload)
}

func MergeConfluenceSection(existing, marker, innerHTML string) string {
	start := "<!-- " + marker + " -->"
	end := "<!-- /" + marker + " -->"
	block := start + innerHTML + end
	if i := strings.Index(existing, start); i >= 0 {
		if j := strings.Index(existing[i:], end); j >= 0 {
			return existing[:i] + block + existing[i+j+len(end):]
		}
	}
	if strings.TrimSpace(existing) == "" {
		return block
	}
	return existing + block
}

func AppendConfluencePage(ctx context.Context, base, email, token, pageRef, marker, heading, report string) (string, error) {
	id, err := ParseConfluencePageID(pageRef)
	if err != nil {
		return "", err
	}
	getURL := strings.TrimRight(base, "/") + "/rest/api/content/" + url.QueryEscape(id) + "?expand=body.storage,version,title,space"
	raw, err := doJSON(ctx, http.MethodGet, getURL, AtlassianAuth(email, token), nil)
	if err != nil {
		return "", err
	}
	var page struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Title string `json:"title"`
		Space struct {
			Key string `json:"key"`
		} `json:"space"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
		Body struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}
	if err := json.Unmarshal([]byte(stripStatus(raw)), &page); err != nil {
		return "", err
	}
	if page.ID == "" {
		page.ID = id
	}
	if page.Type == "" {
		page.Type = "page"
	}
	inner := "<h2>" + html.EscapeString(heading) + "</h2><pre>" + html.EscapeString(truncate(report, 12000)) + "</pre>"
	newBody := MergeConfluenceSection(page.Body.Storage.Value, marker, inner)
	put := map[string]any{
		"id":      page.ID,
		"type":    page.Type,
		"title":   page.Title,
		"version": map[string]any{"number": page.Version.Number + 1},
		"body": map[string]any{
			"storage": map[string]any{
				"value":          newBody,
				"representation": "storage",
			},
		},
	}
	if page.Space.Key != "" {
		put["space"] = map[string]any{"key": page.Space.Key}
	}
	payload, _ := json.Marshal(put)
	putURL := strings.TrimRight(base, "/") + "/rest/api/content/" + url.QueryEscape(page.ID)
	return doJSON(ctx, http.MethodPut, putURL, AtlassianAuth(email, token), payload)
}

func EnsureGitLabMR(ctx context.Context, base, token, projectRef, source, target, title, description string) (string, error) {
	project, err := ParseGitLabProject(projectRef)
	if err != nil {
		return "", err
	}
	id := url.QueryEscape(project)
	api := strings.TrimRight(base, "/") + "/api/v4"
	h := GitLabAuth(token)
	if target == "" {
		raw, err := doJSON(ctx, http.MethodGet, api+"/projects/"+id, h, nil)
		if err == nil {
			var p struct {
				DefaultBranch string `json:"default_branch"`
			}
			_ = json.Unmarshal([]byte(stripStatus(raw)), &p)
			if p.DefaultBranch != "" {
				target = p.DefaultBranch
			}
		}
		if target == "" {
			target = "main"
		}
	}
	body, _ := json.Marshal(map[string]any{
		"source_branch": source,
		"target_branch": target,
		"title":         title,
		"description":   description,
	})
	return doJSON(ctx, http.MethodPost, api+"/projects/"+id+"/merge_requests", h, body)
}

func PushTaskBranch(ctx context.Context, workDir, repo, token, branch string) error {
	return pushTaskBranchAs(ctx, workDir, repo, token, branch, "oauth2")
}

func PushTaskBranchGitHub(ctx context.Context, workDir, repo, token, branch string) error {
	return pushTaskBranchAs(ctx, workDir, repo, token, branch, "x-access-token")
}

func pushTaskBranchAs(ctx context.Context, workDir, repo, token, branch, user string) error {
	if repo == "" || branch == "" {
		return fmt.Errorf("git repo or branch is empty")
	}
	remote := AuthenticatedGitURLAs(repo, token, user)
	dir := filepath.Join(workDir, branch)
	if err := ensureRepo(ctx, remote, dir, branch); err != nil {
		return err
	}
	_ = exec.CommandContext(ctx, "git", "-C", dir, "remote", "set-url", "origin", remote).Run()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "push", "-u", "origin", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push: %s %w", out, err)
	}
	return nil
}

func stripStatus(s string) string {
	if i := strings.Index(s, " body="); i >= 0 {
		return s[i+6:]
	}
	return s
}

func doJSON(ctx context.Context, method, rawURL string, headers map[string]string, body []byte) (string, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, rdr)
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	out := fmt.Sprintf("status=%d body=%s", resp.StatusCode, string(data))
	if resp.StatusCode >= 300 {
		return out, fmt.Errorf("%s", out)
	}
	return out, nil
}
