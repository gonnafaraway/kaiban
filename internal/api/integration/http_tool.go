package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"kaiban/internal/api/textutil"
)

// errorBodySnippet is how much of a failing response we quote back to the model.
const errorBodySnippet = 800

// HTTPTool calls one REST endpoint; path placeholders {name} are filled from args.
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
	if args == "" {
		args = "{}"
	}
	if err := json.Unmarshal([]byte(args), &payload); err != nil {
		return "", errors.Wrap(err, "decode http tool args")
	}
	u, body, err := t.request(payload)
	if err != nil {
		return "", err
	}
	req, err := newJSONRequest(ctx, t.method, u, t.headers, body)
	if err != nil {
		return "", errors.Wrap(err, "build http tool request")
	}
	resp, err := httpClient(defaultHTTPTimeout).Do(req)
	if err != nil {
		return "", errors.Wrap(err, "http tool request")
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := readBody(resp)
	if err != nil {
		return "", errors.Wrap(err, "read http tool body")
	}
	if resp.StatusCode >= 400 {
		return "", errors.Errorf("%s http %d: %s", t.name, resp.StatusCode, textutil.TruncateRunes(data, errorBodySnippet))
	}
	return fmt.Sprintf("status=%d body=%s", resp.StatusCode, data), nil
}

// request resolves URL placeholders and splits leftover args into query or JSON body.
func (t HTTPTool) request(payload map[string]any) (string, []byte, error) {
	u := t.url
	rest := map[string]any{}
	for k, v := range payload {
		ph := "{" + k + "}"
		if strings.Contains(u, ph) {
			u = strings.ReplaceAll(u, ph, url.QueryEscape(fmt.Sprint(v)))
			continue
		}
		rest[k] = v
	}
	if len(rest) == 0 {
		return u, nil, nil
	}
	if t.method == http.MethodGet {
		q := url.Values{}
		for k, v := range rest {
			q.Set(k, fmt.Sprint(v))
		}
		if strings.Contains(u, "?") {
			return u + "&" + q.Encode(), nil, nil
		}
		return u + "?" + q.Encode(), nil, nil
	}
	body, err := marshalJSON(rest, "http tool body")
	if err != nil {
		return "", nil, err
	}
	return u, body, nil
}
