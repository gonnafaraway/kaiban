package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type gitlabProjectMeta struct {
	DefaultBranch string `json:"default_branch"`
}

func GitLabAuth(token string) map[string]string {
	return map[string]string{"PRIVATE-TOKEN": token, "Accept": "application/json"}
}

func GitLabTools(base, token string) []Tool {
	h := GitLabAuth(token)
	base = apiBase(base)
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
