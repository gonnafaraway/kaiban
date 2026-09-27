package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	// defaultHTTPTimeout covers single REST calls to Jira/Confluence/GitLab/GitHub.
	defaultHTTPTimeout = 45 * time.Second
	// maxResponseBytes caps what we read back into tool output.
	maxResponseBytes = 32 * 1024
)

// httpClient is the shared client builder for every integration HTTP call.
func httpClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	return &http.Client{Timeout: timeout}
}

func marshalJSON(v any, what string) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "marshal "+what)
	}
	return b, nil
}

// newJSONRequest builds a JSON request with the given headers.
func newJSONRequest(ctx context.Context, method, rawURL string, headers map[string]string, body []byte) (*http.Request, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, rdr)
	if err != nil {
		return nil, errors.Wrap(err, "build http request")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// readBody reads a capped response body.
func readBody(resp *http.Response) (string, error) {
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", errors.Wrap(err, "read http body")
	}
	return string(data), nil
}

// doJSON performs a REST call and returns "status=<code> body=<payload>".
// Any status >= 300 is also returned as an error, so callers can log the body.
func doJSON(ctx context.Context, method, rawURL string, headers map[string]string, body []byte) (string, error) {
	req, err := newJSONRequest(ctx, method, rawURL, headers, body)
	if err != nil {
		return "", err
	}
	resp, err := httpClient(defaultHTTPTimeout).Do(req)
	if err != nil {
		return "", errors.Wrap(err, "http request")
	}
	defer func() { _ = resp.Body.Close() }()
	payload, err := readBody(resp)
	if err != nil {
		return "", err
	}
	out := fmt.Sprintf("status=%d body=%s", resp.StatusCode, payload)
	if resp.StatusCode >= 300 {
		return out, errors.Errorf("http %d: %s", resp.StatusCode, payload)
	}
	return out, nil
}

// stripStatus drops the "status=… body=" prefix added by doJSON.
func stripStatus(s string) string {
	if i := strings.Index(s, " body="); i >= 0 {
		return s[i+6:]
	}
	return s
}
