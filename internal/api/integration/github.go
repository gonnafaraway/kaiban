package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// GitHubClient is a thin REST client for GitHub / GitHub Enterprise API v3.
type GitHubClient struct {
	BaseURL string
	Token   string
}

type githubRepoMeta struct {
	DefaultBranch string `json:"default_branch"`
}

func GitHubAuth(token string) map[string]string {
	return map[string]string{
		"Authorization":        "Bearer " + token,
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
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

func NewGitHubClient(baseURL, token string) *GitHubClient {
	base := apiBase(baseURL)
	if base == "" {
		base = "https://api.github.com"
	}
	// Enterprise often configured as https://github.example.com — normalize to /api/v3.
	if !strings.Contains(base, "api.github.com") && !strings.HasSuffix(base, "/api/v3") {
		if strings.Contains(base, "github") && !strings.Contains(base, "/api/") {
			base = base + "/api/v3"
		}
	}
	return &GitHubClient{BaseURL: base, Token: token}
}

func (c *GitHubClient) headers() map[string]string {
	return GitHubAuth(c.Token)
}

func (c *GitHubClient) repoURL(owner, repo string) string {
	return c.BaseURL + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
}

// GetRepo fetches repository metadata.
func (c *GitHubClient) GetRepo(ctx context.Context, owner, repo string) (string, error) {
	return doJSON(ctx, http.MethodGet, c.repoURL(owner, repo), c.headers(), nil)
}

// DefaultBranch returns the repository default branch (fallback: main).
func (c *GitHubClient) DefaultBranch(ctx context.Context, owner, repo string) string {
	raw, err := c.GetRepo(ctx, owner, repo)
	if err != nil {
		return "main"
	}
	var p githubRepoMeta
	if err := json.Unmarshal([]byte(stripStatus(raw)), &p); err != nil || p.DefaultBranch == "" {
		return "main"
	}
	return p.DefaultBranch
}

// ListPulls lists pull requests for a repository.
func (c *GitHubClient) ListPulls(ctx context.Context, owner, repo, state string) (string, error) {
	if state == "" {
		state = "open"
	}
	u := c.repoURL(owner, repo) + "/pulls?state=" + url.QueryEscape(state)
	return doJSON(ctx, http.MethodGet, u, c.headers(), nil)
}

// CreatePull creates a pull request. head is the source branch (or owner:branch for forks).
func (c *GitHubClient) CreatePull(ctx context.Context, owner, repo, title, head, base, body string) (string, error) {
	if base == "" {
		base = c.DefaultBranch(ctx, owner, repo)
	}
	payload, err := marshalJSON(map[string]any{
		"title": title,
		"head":  head,
		"base":  base,
		"body":  body,
	}, "github pull")
	if err != nil {
		return "", err
	}
	return doJSON(ctx, http.MethodPost, c.repoURL(owner, repo)+"/pulls", c.headers(), payload)
}

// EnsurePR creates a PR for source → target; returns API response body or error text.
func (c *GitHubClient) EnsurePR(ctx context.Context, projectRef, source, target, title, description string) (string, error) {
	path, err := ParseGitHubRepo(projectRef)
	if err != nil {
		return "", errors.Wrap(err, "parse github repo")
	}
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", errors.Errorf("invalid github repo path %q", path)
	}
	owner, repo := parts[0], parts[1]
	if target == "" {
		target = c.DefaultBranch(ctx, owner, repo)
	}
	return c.CreatePull(ctx, owner, repo, title, source, target, description)
}

// EnsureGitHubPR is a package-level helper used by the usecase.
func EnsureGitHubPR(ctx context.Context, base, token, projectRef, source, target, title, description string) (string, error) {
	return NewGitHubClient(base, token).EnsurePR(ctx, projectRef, source, target, title, description)
}
