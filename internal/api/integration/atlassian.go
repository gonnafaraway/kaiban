package integration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"kaiban/internal/api/textutil"
)

// maxPageChars caps Confluence text we read or write back.
const maxPageChars = 12000

type confluenceWriteArgs struct {
	ID      string `json:"id"`
	Body    string `json:"body"`
	Heading string `json:"heading"`
}

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

// AtlassianAuth builds headers for Jira/Confluence: basic for cloud, bearer for PAT.
func AtlassianAuth(email, token string) map[string]string {
	h := map[string]string{"Accept": "application/json"}
	if email == "" || email == "-" {
		h["Authorization"] = "Bearer " + token
		return h
	}
	h["Authorization"] = basic(email, token)
	return h
}

func basic(email, token string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(email+":"+token))
}

func JiraTools(base, email, token string) []Tool {
	h := AtlassianAuth(email, token)
	base = apiBase(base)
	return []Tool{
		HTTPTool{"jira_search", "Search Jira issues with JQL", http.MethodGet, base + "/rest/api/2/search", h, map[string]any{"type": "object", "properties": map[string]any{"jql": map[string]any{"type": "string"}, "maxResults": map[string]any{"type": "integer"}}}},
		HTTPTool{"jira_get_issue", "Get Jira issue by key", http.MethodGet, base + "/rest/api/2/issue/{issueKey}", h, map[string]any{"type": "object", "properties": map[string]any{"issueKey": map[string]any{"type": "string"}}, "required": []string{"issueKey"}}},
		HTTPTool{"jira_create_issue", "Create Jira issue. Pass Jira REST fields payload.", http.MethodPost, base + "/rest/api/2/issue", h, map[string]any{"type": "object"}},
		HTTPTool{"jira_add_comment", "Add a comment to a Jira issue. Args: issueKey, body (plain text).", http.MethodPost, base + "/rest/api/2/issue/{issueKey}/comment", h, map[string]any{"type": "object", "properties": map[string]any{"issueKey": map[string]any{"type": "string"}, "body": map[string]any{"type": "string"}}, "required": []string{"issueKey", "body"}}},
	}
}

func ConfluenceTools(base, email, token string) []Tool {
	h := AtlassianAuth(email, token)
	base = apiBase(base)
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
	var p confluenceWriteArgs
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", errors.Wrap(err, "decode confluence tool args")
	}
	if t.kind == "comment" {
		htmlBody := "<pre>" + html.EscapeString(p.Body) + "</pre>"
		return PostConfluenceComment(ctx, t.base, t.email, t.token, p.ID, htmlBody)
	}
	if p.Heading == "" {
		p.Heading = "Kaiban"
	}
	return AppendConfluencePage(ctx, t.base, t.email, t.token, p.ID, "kaiban:agent", p.Heading, p.Body)
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
	return textutil.TruncateRunes(text, maxPageChars), nil
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

// MergeConfluenceSection replaces the marked block or appends it, keeping the rest of the page.
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
	inner := "<h2>" + html.EscapeString(heading) + "</h2><pre>" + html.EscapeString(textutil.TruncateRunes(report, maxPageChars)) + "</pre>"
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
