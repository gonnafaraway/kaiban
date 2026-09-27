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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Call(ctx context.Context, args string) (string, error)
}

type HTTPTool struct {
	name, desc, method, url string
	headers                 map[string]string
	params                  map[string]any
}

func (t HTTPTool) Name() string               { return t.name }
func (t HTTPTool) Description() string        { return t.desc }
func (t HTTPTool) Parameters() map[string]any { return t.params }
func (t HTTPTool) Call(ctx context.Context, args string) (string, error) {
	var payload map[string]any
	_ = json.Unmarshal([]byte(args), &payload)
	u := t.url
	bodyPayload := map[string]any{}
	for k, v := range payload {
		ph := "{" + k + "}"
		if strings.Contains(u, ph) {
			u = strings.ReplaceAll(u, ph, url.QueryEscape(fmt.Sprint(v)))
			continue
		}
		bodyPayload[k] = v
	}
	var body io.Reader
	if t.method == http.MethodGet && len(bodyPayload) > 0 {
		q := url.Values{}
		for k, v := range bodyPayload {
			q.Set(k, fmt.Sprint(v))
		}
		if strings.Contains(u, "?") {
			u += "&" + q.Encode()
		} else {
			u += "?" + q.Encode()
		}
	} else if t.method != http.MethodGet && len(bodyPayload) > 0 {
		b, _ := json.Marshal(bodyPayload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, t.method, u, body)
	if err != nil {
		return "", err
	}
	for k, v := range t.headers {
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
	return fmt.Sprintf("status=%d body=%s", resp.StatusCode, string(data)), nil
}

type GitTool struct {
	WorkDir    string
	RepoURL    string
	Branch     string
	Token      string
	GitHubAuth bool
}

func (GitTool) Name() string { return "git_status" }
func (GitTool) Description() string {
	return "Git in the task worktree. Args: {\"command\":\"status|log|diff|commit|push\", \"message\":\"commit message\"}"
}
func (GitTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{"type": "string"},
			"message": map[string]any{"type": "string"},
		},
	}
}

func (g GitTool) Call(ctx context.Context, args string) (string, error) {
	if g.RepoURL == "" {
		return "git repo is not configured", nil
	}
	var p struct {
		Command string `json:"command"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal([]byte(args), &p)
	if p.Command == "" {
		p.Command = "status"
	}
	user := "oauth2"
	if g.GitHubAuth {
		user = "x-access-token"
	}
	repo := AuthenticatedGitURLAs(g.RepoURL, g.Token, user)
	dir := filepath.Join(g.WorkDir, g.Branch)
	if err := ensureRepo(ctx, repo, dir, g.Branch); err != nil {
		return "", err
	}
	switch p.Command {
	case "commit":
		if p.Message == "" {
			p.Message = "kaiban: agent changes"
		}
		if err := safeGitAdd(ctx, dir); err != nil {
			return err.Error(), nil
		}
		cmd := exec.CommandContext(ctx, "git", "-C", dir, "commit", "-m", p.Message)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out) + "\n" + err.Error(), nil
		}
		return string(out), nil
	case "push":
		if err := PushTaskBranch(ctx, g.WorkDir, g.RepoURL, g.Token, g.Branch); err != nil {
			return err.Error(), nil
		}
		return "pushed " + g.Branch, nil
	case "log", "diff", "status":
		cmd := exec.CommandContext(ctx, "git", "-C", dir, p.Command)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out) + "\n" + err.Error(), nil
		}
		return string(out), nil
	default:
		return "unknown git command; use status|log|diff|commit|push", nil
	}
}

func ensureRepo(ctx context.Context, repo, dir, branch string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		_ = exec.CommandContext(ctx, "git", "-C", dir, "remote", "set-url", "origin", repo).Run()
		_ = exec.CommandContext(ctx, "git", "-C", dir, "fetch", "origin", "--prune").Run()
		// Prefer existing local branch (keep agent commits). Only create from origin when missing.
		if err := exec.CommandContext(ctx, "git", "-C", dir, "checkout", branch).Run(); err != nil {
			if err := exec.CommandContext(ctx, "git", "-C", dir, "checkout", "-B", branch, "origin/"+branch).Run(); err != nil {
				_ = exec.CommandContext(ctx, "git", "-C", dir, "checkout", "-B", branch).Run()
			}
		}
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(dir), 0o755)
	cmd := exec.CommandContext(ctx, "git", "clone", "--branch", branch, "--single-branch", repo, dir)
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(dir)
		cmd = exec.CommandContext(ctx, "git", "clone", repo, dir)
		if out, err2 := cmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("clone: %s %w", out, err2)
		}
		if err := exec.CommandContext(ctx, "git", "-C", dir, "checkout", "-B", branch, "origin/"+branch).Run(); err != nil {
			_ = exec.CommandContext(ctx, "git", "-C", dir, "checkout", "-B", branch).Run()
		}
	}
	return nil
}

// RefreshTaskWorktree fetches remotes and updates base ref so diff/approve use fresh data.
func RefreshTaskWorktree(ctx context.Context, workDir, repoURL, branch, base string) error {
	if repoURL == "" || branch == "" {
		return nil
	}
	dir := filepath.Join(workDir, branch)
	if err := ensureRepo(ctx, repoURL, dir, branch); err != nil {
		return err
	}
	if base != "" && base != branch {
		// Force-update local base tip even if history was rewritten.
		_ = exec.CommandContext(ctx, "git", "-C", dir, "fetch", "origin", "+"+base+":"+base).Run()
	}
	return nil
}

func EnsureTaskBranch(ctx context.Context, workDir, repoURL, defaultBranch, taskBranch string) error {
	return RefreshTaskWorktree(ctx, workDir, repoURL, taskBranch, defaultBranch)
}

var secretPathFragments = []string{
	".env", ".pem", ".key", "id_rsa", "id_ed25519", "credentials", "secret",
	".npmrc", ".pypirc", "kubeconfig", "service-account",
}

func isSecretPath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	full := strings.ToLower(filepath.ToSlash(path))
	for _, frag := range secretPathFragments {
		if strings.Contains(base, frag) || strings.Contains(full, "/"+frag) {
			return true
		}
	}
	return false
}

func safeGitAdd(ctx context.Context, dir string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain", "-u").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git status: %s %w", out, err)
	}
	var added, skipped int
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if path == "" {
			continue
		}
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}
		path = strings.Trim(path, "\"")
		if isSecretPath(path) {
			skipped++
			continue
		}
		if err := exec.CommandContext(ctx, "git", "-C", dir, "add", "--", path).Run(); err != nil {
			continue
		}
		added++
	}
	if added == 0 && skipped > 0 {
		return fmt.Errorf("refused to stage %d secret-looking path(s); nothing left to commit", skipped)
	}
	return nil
}

func JiraTools(base, email, token string) []Tool {
	h := AtlassianAuth(email, token)
	base = strings.TrimRight(base, "/")
	return []Tool{
		HTTPTool{"jira_search", "Search Jira issues with JQL", http.MethodGet, base + "/rest/api/2/search", h, map[string]any{"type": "object", "properties": map[string]any{"jql": map[string]any{"type": "string"}, "maxResults": map[string]any{"type": "integer"}}}},
		HTTPTool{"jira_get_issue", "Get Jira issue by key", http.MethodGet, base + "/rest/api/2/issue/{issueKey}", h, map[string]any{"type": "object", "properties": map[string]any{"issueKey": map[string]any{"type": "string"}}, "required": []string{"issueKey"}}},
		HTTPTool{"jira_create_issue", "Create Jira issue. Pass Jira REST fields payload.", http.MethodPost, base + "/rest/api/2/issue", h, map[string]any{"type": "object"}},
		HTTPTool{"jira_add_comment", "Add a comment to a Jira issue. Args: issueKey, body (plain text).", http.MethodPost, base + "/rest/api/2/issue/{issueKey}/comment", h, map[string]any{"type": "object", "properties": map[string]any{"issueKey": map[string]any{"type": "string"}, "body": map[string]any{"type": "string"}}, "required": []string{"issueKey", "body"}}},
	}
}

func ConfluenceTools(base, email, token string) []Tool {
	h := AtlassianAuth(email, token)
	base = strings.TrimRight(base, "/")
	return []Tool{
		HTTPTool{"confluence_search", "Search Confluence CQL", http.MethodGet, base + "/rest/api/content/search", h, map[string]any{"type": "object", "properties": map[string]any{"cql": map[string]any{"type": "string"}}}},
		HTTPTool{"confluence_get_page", "Get Confluence page by id. Expand body.", http.MethodGet, base + "/rest/api/content/{id}?expand=body.storage,version", h, map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}}},
		confluenceWriteTool{kind: "comment", base: base, email: email, token: token},
		confluenceWriteTool{kind: "page", base: base, email: email, token: token},
	}
}

type confluenceWriteTool struct {
	kind, base, email, token string
}

func (t confluenceWriteTool) Name() string {
	if t.kind == "comment" {
		return "confluence_add_comment"
	}
	return "confluence_update_page"
}

func (t confluenceWriteTool) Description() string {
	if t.kind == "comment" {
		return "Add a footer comment to a Confluence page. Args: id (page id or URL), body (text or HTML)."
	}
	return "Write a Kaiban section onto a Confluence page body (does not wipe the rest of the page). Args: id (page id or URL), heading, body."
}

func (t confluenceWriteTool) Parameters() map[string]any {
	props := map[string]any{
		"id":   map[string]any{"type": "string"},
		"body": map[string]any{"type": "string"},
	}
	req := []string{"id", "body"}
	if t.kind != "comment" {
		props["heading"] = map[string]any{"type": "string"}
	}
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func (t confluenceWriteTool) Call(ctx context.Context, args string) (string, error) {
	var p struct {
		ID      string `json:"id"`
		Body    string `json:"body"`
		Heading string `json:"heading"`
	}
	_ = json.Unmarshal([]byte(args), &p)
	if t.kind == "comment" {
		htmlBody := "<pre>" + html.EscapeString(p.Body) + "</pre>"
		return PostConfluenceComment(ctx, t.base, t.email, t.token, p.ID, htmlBody)
	}
	if p.Heading == "" {
		p.Heading = "Kaiban"
	}
	return AppendConfluencePage(ctx, t.base, t.email, t.token, p.ID, "kaiban:agent", p.Heading, p.Body)
}

func GitLabTools(base, token string) []Tool {
	h := GitLabAuth(token)
	base = strings.TrimRight(base, "/")
	return []Tool{
		HTTPTool{"gitlab_get_project", "Get GitLab project by URL-encoded path id.", http.MethodGet, base + "/api/v4/projects/{id}", h, map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}}},
		HTTPTool{"gitlab_list_mrs", "List merge requests of a project.", http.MethodGet, base + "/api/v4/projects/{id}/merge_requests", h, map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}}},
		HTTPTool{"gitlab_create_mr", "Create merge request. Args: id (project path), source_branch, target_branch, title, description.", http.MethodPost, base + "/api/v4/projects/{id}/merge_requests", h, map[string]any{"type": "object", "properties": map[string]any{
			"id":            map[string]any{"type": "string"},
			"source_branch": map[string]any{"type": "string"},
			"target_branch": map[string]any{"type": "string"},
			"title":         map[string]any{"type": "string"},
			"description":   map[string]any{"type": "string"},
		}, "required": []string{"id", "source_branch", "title"}}},
	}
}

func GitHubTools(base, token string) []Tool {
	c := NewGitHubClient(base, token)
	h := c.headers()
	api := c.BaseURL
	return []Tool{
		HTTPTool{"github_get_repo", "Get GitHub repository. Args: owner, repo.", http.MethodGet, api + "/repos/{owner}/{repo}", h, map[string]any{"type": "object", "properties": map[string]any{
			"owner": map[string]any{"type": "string"},
			"repo":  map[string]any{"type": "string"},
		}, "required": []string{"owner", "repo"}}},
		HTTPTool{"github_list_prs", "List pull requests. Args: owner, repo, optional state (open|closed|all).", http.MethodGet, api + "/repos/{owner}/{repo}/pulls", h, map[string]any{"type": "object", "properties": map[string]any{
			"owner": map[string]any{"type": "string"},
			"repo":  map[string]any{"type": "string"},
			"state": map[string]any{"type": "string"},
		}, "required": []string{"owner", "repo"}}},
		HTTPTool{"github_create_pr", "Create a pull request. Args: owner, repo, title, head (source branch), base (target branch), body.", http.MethodPost, api + "/repos/{owner}/{repo}/pulls", h, map[string]any{"type": "object", "properties": map[string]any{
			"owner": map[string]any{"type": "string"},
			"repo":  map[string]any{"type": "string"},
			"title": map[string]any{"type": "string"},
			"head":  map[string]any{"type": "string"},
			"base":  map[string]any{"type": "string"},
			"body":  map[string]any{"type": "string"},
		}, "required": []string{"owner", "repo", "title", "head"}}},
	}
}

func basic(email, token string) string {
	raw := email + ":" + token
	return "Basic " + b64(raw)
}

func b64(s string) string {
	return stdB64(s)
}

// MCP JSON-RPC initialize + tools/list (streamable HTTP / JSON)
func HandshakeMCP(ctx context.Context, endpoint string, headers map[string]string) (map[string]any, []Tool, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	initBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "kaiban", "version": "0.1.0"},
		},
	}
	if _, err := mcpPost(ctx, client, endpoint, headers, initBody); err != nil {
		return nil, nil, err
	}
	listBody := map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": map[string]any{}}
	raw, err := mcpPost(ctx, client, endpoint, headers, listBody)
	if err != nil {
		return nil, nil, err
	}
	var parsed struct {
		Result struct {
			Tools []struct {
				Name        string         `json:"name"`
				Description string         `json:"description"`
				InputSchema map[string]any `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return map[string]any{"raw": string(raw)}, nil, nil
	}
	if parsed.Error != nil {
		return nil, nil, fmt.Errorf("mcp: %s", parsed.Error.Message)
	}
	var tools []Tool
	caps := map[string]any{"tools": parsed.Result.Tools}
	for _, t := range parsed.Result.Tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		tools = append(tools, mcpTool{name: t.Name, desc: t.Description, schema: schema, endpoint: endpoint, headers: headers, client: client})
	}
	return caps, tools, nil
}

type mcpTool struct {
	name, desc, endpoint string
	schema               map[string]any
	headers              map[string]string
	client               *http.Client
}

func (m mcpTool) Name() string               { return m.name }
func (m mcpTool) Description() string        { return m.desc }
func (m mcpTool) Parameters() map[string]any { return m.schema }
func (m mcpTool) Call(ctx context.Context, args string) (string, error) {
	var arg any
	_ = json.Unmarshal([]byte(args), &arg)
	body := map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "tools/call",
		"params": map[string]any{"name": m.name, "arguments": arg},
	}
	raw, err := mcpPost(ctx, m.client, m.endpoint, m.headers, body)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func mcpPost(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, body any) ([]byte, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mcp http %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}
