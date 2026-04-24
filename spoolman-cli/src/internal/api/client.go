package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/vibecoder/spoolctl/internal/config"
)

type Client struct {
	baseURL string
	http    *http.Client
	verbose bool
}

type APIError struct {
	Status  int             `json:"status"`
	Message string          `json:"error"`
	Detail  json.RawMessage `json:"detail,omitempty"`
}

func (e *APIError) Error() string {
	if e.Detail != nil {
		return fmt.Sprintf("HTTP %d: %s (detail: %s)", e.Status, e.Message, e.Detail)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
}

func NewClient(cfg *config.Config, verbose bool) (*Client, error) {
	hc, err := cfg.HTTPClient()
	if err != nil {
		return nil, err
	}
	return &Client{baseURL: cfg.ServerURL, http: hc, verbose: verbose}, nil
}

func (c *Client) get(path string, query url.Values, out interface{}) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	resp, err := c.http.Get(u)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	return c.decode(resp, out)
}

func (c *Client) post(path string, body interface{}, out interface{}) error {
	return c.doJSON(http.MethodPost, path, body, out)
}

func (c *Client) patch(path string, body interface{}, out interface{}) error {
	return c.doJSON(http.MethodPatch, path, body, out)
}

func (c *Client) put(path string, body interface{}, out interface{}) error {
	return c.doJSON(http.MethodPut, path, body, out)
}

func (c *Client) delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.errorFrom(resp)
	}
	return nil
}

func (c *Client) doJSON(method, path string, body interface{}, out interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return err
	}
	req, err := http.NewRequest(method, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	return c.decode(resp, out)
}

func (c *Client) decode(resp *http.Response, out interface{}) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.errorFrom(resp)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) errorFrom(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	// Try to extract a detail message from FastAPI's standard error format
	var fe struct {
		Detail interface{} `json:"detail"`
	}
	msg := http.StatusText(resp.StatusCode)
	var detail json.RawMessage
	if json.Unmarshal(body, &fe) == nil && fe.Detail != nil {
		detailBytes, _ := json.Marshal(fe.Detail)
		detail = detailBytes
		if s, ok := fe.Detail.(string); ok {
			msg = s
		}
	}
	return &APIError{Status: resp.StatusCode, Message: msg, Detail: detail}
}

// RawGet returns the raw JSON bytes for a GET request (for pass-through output).
func (c *Client) RawGet(path string, query url.Values) (json.RawMessage, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	resp, err := c.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.errorFrom(resp)
	}
	return io.ReadAll(resp.Body)
}
