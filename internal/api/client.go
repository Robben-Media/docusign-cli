package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	userAgent  string
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

func WithUserAgent(ua string) ClientOption {
	return func(c *Client) {
		c.userAgent = ua
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:    apiKey,
		userAgent: "placeholder-cli/1.0",
		baseURL:   "https://api.example.com",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type Request struct {
	Method  string
	Path    string
	Body    any
	Headers map[string]string
}

func (c *Client) Do(ctx context.Context, req Request) (*http.Response, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := c.baseURL + req.Path
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)

	// Set API key header (override in specific CLI implementations)
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}

func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.doJSON(ctx, Request{Method: http.MethodGet, Path: path}, result)
}

func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.doJSON(ctx, Request{Method: http.MethodPost, Path: path, Body: body}, result)
}

func (c *Client) Put(ctx context.Context, path string, body, result any) error {
	return c.doJSON(ctx, Request{Method: http.MethodPut, Path: path, Body: body}, result)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.Do(ctx, Request{Method: http.MethodDelete, Path: path})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}

	return nil
}

func (c *Client) doJSON(ctx context.Context, req Request, result any) error {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}

func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var apiErr struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	// Try to parse as JSON
	if json.Unmarshal(body, &apiErr) == nil {
		msg := apiErr.Message
		if msg == "" {
			msg = apiErr.Error
		}
		if msg != "" {
			return &APIError{StatusCode: resp.StatusCode, Message: msg}
		}
	}

	// Fallback to status text
	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    http.StatusText(resp.StatusCode),
	}
}
