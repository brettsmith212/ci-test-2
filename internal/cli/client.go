package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an HTTP client for API communication
type Client struct {
	httpClient *http.Client
	config     *Config
}

// NewClient creates a new API client
func NewClient(config *Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// Request represents an HTTP request
type Request struct {
	Method string
	Path   string
	Body   interface{}
	Headers map[string]string
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do performs an HTTP request
func (c *Client) Do(req Request) (*Response, error) {
	// Prepare request body
	var body io.Reader
	if req.Body != nil {
		jsonData, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	// Create HTTP request
	url := c.config.GetAPIEndpoint(req.Path)
	httpReq, err := http.NewRequest(req.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "ampx-cli/1.0")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Perform request
	if c.config.Verbose {
		fmt.Printf("Making %s request to %s\n", req.Method, url)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.config.Verbose {
		fmt.Printf("Response status: %d\n", resp.StatusCode)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}, nil
}

// Get performs a GET request
func (c *Client) Get(path string) (*Response, error) {
	return c.Do(Request{
		Method: "GET",
		Path:   path,
	})
}

// Post performs a POST request
func (c *Client) Post(path string, body interface{}) (*Response, error) {
	return c.Do(Request{
		Method: "POST",
		Path:   path,
		Body:   body,
	})
}

// Patch performs a PATCH request
func (c *Client) Patch(path string, body interface{}) (*Response, error) {
	return c.Do(Request{
		Method: "PATCH",
		Path:   path,
		Body:   body,
	})
}

// Delete performs a DELETE request
func (c *Client) Delete(path string) (*Response, error) {
	return c.Do(Request{
		Method: "DELETE",
		Path:   path,
	})
}

// CheckHealth checks if the API server is healthy
func (c *Client) CheckHealth() error {
	resp, err := c.Get("/health")
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API server is not healthy (status: %d)", resp.StatusCode)
	}

	return nil
}

// Ping tests connectivity to the API server
func (c *Client) Ping() error {
	resp, err := c.Get("/api/v1/ping")
	if err != nil {
		return fmt.Errorf("failed to ping API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed (status: %d)", resp.StatusCode)
	}

	// Parse response
	var pingResp map[string]interface{}
	if err := json.Unmarshal(resp.Body, &pingResp); err != nil {
		return fmt.Errorf("failed to parse ping response: %w", err)
	}

	if message, ok := pingResp["message"].(string); ok && message == "pong" {
		return nil
	}

	return fmt.Errorf("unexpected ping response: %s", string(resp.Body))
}

// APIError represents an error returned by the API
type APIError struct {
	Type      string            `json:"error"`
	Message   string            `json:"message"`
	Details   string            `json:"details,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// Error implements the error interface
func (e APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ParseError attempts to parse an API error from a response
func (c *Client) ParseError(resp *Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Try to parse as API error
	var apiErr APIError
	if err := json.Unmarshal(resp.Body, &apiErr); err != nil {
		// Not a structured API error, return generic error
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(resp.Body))
	}

	return apiErr
}

// HandleResponse is a helper function to handle common response patterns
func (c *Client) HandleResponse(resp *Response, target interface{}) error {
	// Check for errors
	if err := c.ParseError(resp); err != nil {
		return err
	}

	// If target is nil, we don't need to parse the body
	if target == nil {
		return nil
	}

	// Parse response body into target
	if err := json.Unmarshal(resp.Body, target); err != nil {
		return fmt.Errorf("failed to parse response body: %w", err)
	}

	return nil
}

// IsConnectable checks if the API server is reachable
func (c *Client) IsConnectable() bool {
	err := c.Ping()
	return err == nil
}

// GetVersion gets the API version information
func (c *Client) GetVersion() (map[string]interface{}, error) {
	resp, err := c.Get("/api/v1/ping")
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	var result map[string]interface{}
	if err := c.HandleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}
