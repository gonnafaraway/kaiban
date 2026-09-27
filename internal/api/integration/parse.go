package integration

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var jiraKeyRe = regexp.MustCompile(`([A-Z][A-Z0-9]+-\d+)`)

func ParseJiraIssue(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("jira issue is empty")
	}
	if m := jiraKeyRe.FindString(raw); m != "" {
		return m, nil
	}
	return "", fmt.Errorf("cannot parse jira issue from %q", raw)
}

func ParseConfluencePageID(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("confluence url is empty")
	}
	if u, err := url.Parse(raw); err == nil {
		if id := u.Query().Get("pageId"); id != "" {
			return id, nil
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if last != "" && isDigits(last) {
				return last, nil
			}
		}
	}
	if isDigits(raw) {
		return raw, nil
	}
	return "", fmt.Errorf("cannot parse confluence page id from %q", raw)
}

func ParseGitLabProject(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	if raw == "" {
		return "", fmt.Errorf("gitlab repo is empty")
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" && u.Path != "" {
		p := strings.Trim(u.Path, "/")
		p = strings.TrimSuffix(p, ".git")
		if p != "" {
			return p, nil
		}
	}
	if strings.Contains(raw, "/") && !strings.Contains(raw, "://") {
		return strings.Trim(raw, "/"), nil
	}
	return "", fmt.Errorf("cannot parse gitlab project from %q", raw)
}

// ParseGitHubRepo returns "owner/repo" from a URL, SSH form, or path.
func ParseGitHubRepo(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	if raw == "" {
		return "", fmt.Errorf("github repo is empty")
	}
	if strings.HasPrefix(raw, "git@") {
		// git@github.com:owner/repo
		if i := strings.Index(raw, ":"); i >= 0 {
			p := strings.Trim(raw[i+1:], "/")
			p = strings.TrimSuffix(p, ".git")
			if strings.Count(p, "/") >= 1 {
				return p, nil
			}
		}
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" && u.Path != "" {
		p := strings.Trim(u.Path, "/")
		p = strings.TrimSuffix(p, ".git")
		parts := strings.Split(p, "/")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return parts[0] + "/" + parts[1], nil
		}
	}
	if strings.Contains(raw, "/") && !strings.Contains(raw, "://") {
		p := strings.Trim(raw, "/")
		parts := strings.Split(p, "/")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return parts[0] + "/" + parts[1], nil
		}
	}
	return "", fmt.Errorf("cannot parse github repo from %q", raw)
}

func IsGitHubHost(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "git@github.com:") {
		return true
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		h := strings.ToLower(u.Host)
		return h == "github.com" || strings.HasSuffix(h, ".ghe.com") || strings.Contains(h, "github")
	}
	return strings.Contains(raw, "github.com/") || strings.HasPrefix(raw, "github.com/")
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func AtlassianAuth(email, token string) map[string]string {
	h := map[string]string{"Accept": "application/json"}
	email = strings.TrimSpace(email)
	if email == "" || email == "-" {
		h["Authorization"] = "Bearer " + token
		return h
	}
	h["Authorization"] = basic(email, token)
	return h
}

func GitLabAuth(token string) map[string]string {
	return map[string]string{"PRIVATE-TOKEN": token, "Accept": "application/json"}
}

func GitHubAuth(token string) map[string]string {
	return map[string]string{
		"Authorization":        "Bearer " + token,
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	}
}

func AuthenticatedGitURL(repo, token string) string {
	return AuthenticatedGitURLAs(repo, token, "oauth2")
}

func AuthenticatedGitHubURL(repo, token string) string {
	return AuthenticatedGitURLAs(repo, token, "x-access-token")
}

func AuthenticatedGitURLAs(repo, token, user string) string {
	if token == "" || !strings.Contains(repo, "://") {
		return repo
	}
	u, err := url.Parse(repo)
	if err != nil {
		return repo
	}
	if user == "" {
		user = "oauth2"
	}
	u.User = url.UserPassword(user, token)
	return u.String()
}
