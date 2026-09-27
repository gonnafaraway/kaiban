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

	"github.com/pkg/errors"
)

type confluenceStorageBody struct {
	Value string `json:"value"`
}

type confluencePageBody struct {
	Storage confluenceStorageBody `json:"storage"`
}

type confluenceSpaceRef struct {
	Key string `json:"key"`
}

type confluenceVersion struct {
	Number int `json:"number"`
}

type confluencePageSnippet struct {
	Title string             `json:"title"`
	Body  confluencePageBody `json:"body"`
}

type confluencePageDoc struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Title   string             `json:"title"`
	Space   confluenceSpaceRef `json:"space"`
	Version confluenceVersion  `json:"version"`
	Body    confluencePageBody `json:"body"`
}

type gitlabProjectMeta struct {
	DefaultBranch string `json:"default_branch"`
}

func PostJiraComment(ctx context.Context, base, email, token, issue, text string) (string, error) {
	key, err := ParseJiraIssue(issue)
	if err != nil {
		return "", errors.Wrap(err, "parse jira issue")
	}
	u := apiBase(base) + "/rest/api/2/issue/" + url.QueryEscape(key) + "/comment"
	payload, err := marshalJSON(map[string]any{"body": text}, "jira comment")
	if err != nil {
		return "", err
	}
	out, err := doJSON(ctx, http.MethodPost, u, AtlassianAuth(email, token), payload)
	if err != nil {
		return "", errors.Wrap(err, "post jira comment")
	}
	return out, nil
}

func FetchConfluencePage(ctx context.Context, base, email, token, pageRef string) (string, error) {
	id, err := ParseConfluencePageID(pageRef)
	if err != nil {
		return "", errors.Wrap(err, "parse confluence page id")
	}
	u := apiBase(base) + "/rest/api/content/" + url.QueryEscape(id) + "?expand=body.storage,version,title"
	raw, err := doJSON(ctx, http.MethodGet, u, AtlassianAuth(email, token), nil)
	if err != nil {
		return "", errors.Wrap(err, "fetch confluence page")
	}
	var parsed confluencePageSnippet
	if err := json.Unmarshal([]byte(stripStatus(raw)), &parsed); err != nil {
		return "", errors.Wrap(err, "decode confluence page")
	}
	text := parsed.Title + "\n" + parsed.Body.Storage.Value
	return truncate(text, 12000), nil
}

func PostConfluenceComment(ctx context.Context, base, email, token, pageRef, htmlBody string) (string, error) {
	id, err := ParseConfluencePageID(pageRef)
	if err != nil {
		return "", errors.Wrap(err, "parse confluence page id")
	}
	u := apiBase(base) + "/rest/api/content"
	payload, err := marshalJSON(map[string]any{
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
	}, "confluence comment")
	if err != nil {
		return "", err
	}
	out, err := doJSON(ctx, http.MethodPost, u, AtlassianAuth(email, token), payload)
	if err != nil {
		return "", errors.Wrap(err, "post confluence comment")
	}
	return out, nil
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
	if existing == "" {
		return block
	}
	return existing + block
}

func AppendConfluencePage(ctx context.Context, base, email, token, pageRef, marker, heading, report string) (string, error) {
	id, err := ParseConfluencePageID(pageRef)
	if err != nil {
		return "", errors.Wrap(err, "parse confluence page id")
	}
	getURL := apiBase(base) + "/rest/api/content/" + url.QueryEscape(id) + "?expand=body.storage,version,title,space"
	raw, err := doJSON(ctx, http.MethodGet, getURL, AtlassianAuth(email, token), nil)
	if err != nil {
		return "", errors.Wrap(err, "get confluence page")
	}
	var page confluencePageDoc
	if err := json.Unmarshal([]byte(stripStatus(raw)), &page); err != nil {
		return "", errors.Wrap(err, "decode confluence page")
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
	payload, err := marshalJSON(put, "confluence page")
	if err != nil {
		return "", err
	}
	putURL := apiBase(base) + "/rest/api/content/" + url.QueryEscape(page.ID)
	out, err := doJSON(ctx, http.MethodPut, putURL, AtlassianAuth(email, token), payload)
	if err != nil {
		return "", errors.Wrap(err, "update confluence page")
	}
	return out, nil
}

func EnsureGitLabMR(ctx context.Context, base, token, projectRef, source, target, title, description string) (string, error) {
	project, err := ParseGitLabProject(projectRef)
	if err != nil {
		return "", errors.Wrap(err, "parse gitlab project")
	}
	id := url.QueryEscape(project)
	api := apiBase(base) + "/api/v4"
	h := GitLabAuth(token)
	if target == "" {
		raw, err := doJSON(ctx, http.MethodGet, api+"/projects/"+id, h, nil)
		if err == nil {
			var p gitlabProjectMeta
			if uerr := json.Unmarshal([]byte(stripStatus(raw)), &p); uerr == nil && p.DefaultBranch != "" {
				target = p.DefaultBranch
			}
		}
		if target == "" {
			target = "main"
		}
	}
	body, err := marshalJSON(map[string]any{
		"source_branch": source,
		"target_branch": target,
		"title":         title,
		"description":   description,
	}, "gitlab mr")
	if err != nil {
		return "", err
	}
	out, err := doJSON(ctx, http.MethodPost, api+"/projects/"+id+"/merge_requests", h, body)
	if err != nil {
		return "", errors.Wrap(err, "create gitlab mr")
	}
	return out, nil
}

func PushTaskBranch(ctx context.Context, workDir, repo, token, branch string) error {
	return pushTaskBranchAs(ctx, workDir, repo, token, branch, "oauth2")
}

func PushTaskBranchGitHub(ctx context.Context, workDir, repo, token, branch string) error {
	return pushTaskBranchAs(ctx, workDir, repo, token, branch, "x-access-token")
}

func pushTaskBranchAs(ctx context.Context, workDir, repo, token, branch, user string) error {
	if repo == "" || branch == "" {
		return errors.New("git repo or branch is empty")
	}
	remote := AuthenticatedGitURLAs(repo, token, user)
	dir := filepath.Join(workDir, branch)
	if err := ensureRepo(ctx, remote, dir, branch); err != nil {
		return errors.Wrap(err, "ensure git repo")
	}
	_ = exec.CommandContext(ctx, "git", "-C", dir, "remote", "set-url", "origin", remote).Run()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "push", "-u", "origin", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git push: %s", out)
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
		return "", errors.Wrap(err, "build http request")
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
		return "", errors.Wrap(err, "http request")
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if err != nil {
		return "", errors.Wrap(err, "read http body")
	}
	out := fmt.Sprintf("status=%d body=%s", resp.StatusCode, string(data))
	if resp.StatusCode >= 300 {
		return out, errors.Errorf("http %d: %s", resp.StatusCode, string(data))
	}
	return out, nil
}
