package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DiffFile is one path changed vs base.
type DiffFile struct {
	Path       string `json:"path"`
	Insertions int    `json:"insertions"`
	Deletions  int    `json:"deletions"`
}

// DiffSummary is a compact view of task branch vs default.
type DiffSummary struct {
	Base       string     `json:"base"`
	Branch     string     `json:"branch"`
	Files      []DiffFile `json:"files"`
	Insertions int        `json:"insertions"`
	Deletions  int        `json:"deletions"`
	Commits    []string   `json:"commits"`
	Empty      bool       `json:"empty"`
}

// TaskDiffSummary runs git diff --numstat and log against base..branch in the task worktree.
func TaskDiffSummary(ctx context.Context, workDir, branch, base string) (*DiffSummary, error) {
	if branch == "" {
		return &DiffSummary{Empty: true}, nil
	}
	if base == "" {
		base = "main"
	}
	dir := filepath.Join(workDir, branch)
	out := &DiffSummary{Base: base, Branch: branch, Files: []DiffFile{}, Commits: []string{}}

	numstat, err := exec.CommandContext(ctx, "git", "-C", dir, "diff", "--numstat", base+"..."+branch).CombinedOutput()
	if err != nil {
		// try without three-dot if base missing locally
		numstat, err = exec.CommandContext(ctx, "git", "-C", dir, "diff", "--numstat", base).CombinedOutput()
		if err != nil {
			return out, fmt.Errorf("git diff: %s %w", strings.TrimSpace(string(numstat)), err)
		}
	}
	for _, line := range strings.Split(string(numstat), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		ins, _ := strconv.Atoi(parts[0])
		del, _ := strconv.Atoi(parts[1])
		path := parts[2]
		if parts[0] == "-" {
			ins = 0
		}
		if parts[1] == "-" {
			del = 0
		}
		out.Files = append(out.Files, DiffFile{Path: path, Insertions: ins, Deletions: del})
		out.Insertions += ins
		out.Deletions += del
	}

	logOut, _ := exec.CommandContext(ctx, "git", "-C", dir, "log", "--oneline", "-n", "20", base+".."+branch).CombinedOutput()
	for _, line := range strings.Split(string(logOut), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out.Commits = append(out.Commits, line)
		}
	}
	out.Empty = len(out.Files) == 0 && len(out.Commits) == 0
	return out, nil
}

// TaskDiffUnified returns truncated unified diff text.
func TaskDiffUnified(ctx context.Context, workDir, branch, base string, maxBytes int) (string, error) {
	if branch == "" {
		return "", nil
	}
	if base == "" {
		base = "main"
	}
	if maxBytes <= 0 {
		maxBytes = 200 * 1024
	}
	dir := filepath.Join(workDir, branch)
	raw, err := exec.CommandContext(ctx, "git", "-C", dir, "diff", base+"..."+branch).CombinedOutput()
	if err != nil {
		raw, err = exec.CommandContext(ctx, "git", "-C", dir, "diff", base).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git diff: %s %w", strings.TrimSpace(string(raw)), err)
		}
	}
	s := string(raw)
	if len(s) > maxBytes {
		s = s[:maxBytes] + "\n… (truncated)"
	}
	return s, nil
}

// ExtractRemoteURL pulls html_url (GitHub) or web_url (GitLab) from Ensure* response bodies.
func ExtractRemoteURL(raw string) string {
	body := stripStatus(raw)
	var p struct {
		HTMLURL string `json:"html_url"`
		WebURL  string `json:"web_url"`
	}
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		return ""
	}
	if p.HTMLURL != "" {
		return p.HTMLURL
	}
	return p.WebURL
}
