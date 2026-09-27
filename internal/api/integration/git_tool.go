package integration

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type gitToolArgs struct {
	Command string `json:"command"`
	Message string `json:"message"`
}

// GitTool gives the agent git access inside the task worktree.
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
	var p gitToolArgs
	if args == "" {
		args = "{}"
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", errors.Wrap(err, "decode git tool args")
	}
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
		return "", errors.Wrap(err, "ensure git repo")
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
			return errors.Wrapf(err2, "clone: %s", out)
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
		return errors.Wrap(err, "ensure git repo")
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
		return errors.Wrapf(err, "git status: %s", out)
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
		return errors.Errorf("refused to stage %d secret-looking path(s); nothing left to commit", skipped)
	}
	return nil
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
