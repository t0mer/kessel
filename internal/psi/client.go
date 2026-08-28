package psi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultEndpoint is the PageSpeed Insights v5 runPagespeed endpoint.
const DefaultEndpoint = "https://www.googleapis.com/pagespeedonline/v5/runPagespeed"

// Client calls the PageSpeed Insights API.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	maxRetries int
	backoff    func(attempt int) time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the PSI API key (keyless when empty).
func WithAPIKey(key string) Option { return func(c *Client) { c.apiKey = key } }

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// WithBaseURL overrides the API endpoint (test hook).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithRetries sets the retry count and backoff.
func WithRetries(max int, backoff func(int) time.Duration) Option {
	return func(c *Client) { c.maxRetries = max; c.backoff = backoff }
}

// NewClient builds a Client with sensible defaults.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    DefaultEndpoint,
		maxRetries: 3,
		backoff:    func(attempt int) time.Duration { return time.Duration(1<<attempt) * time.Second },
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

var categories = []string{"performance", "accessibility", "best-practices", "seo"}

func (c *Client) buildURL(targetURL, strategy string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parsing base URL: %w", err)
	}
	q := u.Query()
	q.Set("url", targetURL)
	q.Set("strategy", strategy)
	for _, cat := range categories {
		q.Add("category", cat)
	}
	if c.apiKey != "" {
		q.Set("key", c.apiKey)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Run executes a PSI check for targetURL with the given strategy
// ("mobile" or "desktop"), returning the parsed result and raw JSON body.
func (c *Client) Run(ctx context.Context, targetURL, strategy string) (*Result, []byte, error) {
	reqURL, err := c.buildURL(targetURL, strategy)
	if err != nil {
		return nil, nil, err
	}
	body, err := c.doWithRetry(ctx, reqURL)
	if err != nil {
		return nil, nil, err
	}
	res, err := parseResult(body)
	if err != nil {
		return nil, body, err
	}
	return res, body, nil
}

func (c *Client) do(ctx context.Context, reqURL string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("PSI request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("reading PSI body: %w", err)
	}
	return resp.StatusCode, body, nil
}

// APIError is a non-2xx response from the PSI API.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("PSI API returned status %d: %s", e.Status, truncate(e.Body, 200))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func retryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func (c *Client) doWithRetry(ctx context.Context, reqURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.backoff(attempt - 1)):
			}
		}
		status, body, err := c.do(ctx, reqURL)
		if err != nil {
			lastErr = err
			continue // transport error: retry
		}
		if status >= 200 && status < 300 {
			return body, nil
		}
		lastErr = &APIError{Status: status, Body: string(body)}
		if !retryable(status) {
			return nil, lastErr
		}
	}
	return nil, fmt.Errorf("PSI request failed after %d retries: %w", c.maxRetries, lastErr)
}
