// Package http provides a client for making HTTP requests
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default timeout for HTTP requests.
	DefaultTimeout = 30 * time.Second
)

type (
	// Request is a struct that contains the data needed to make an HTTP request.
	Request struct {
		Path            string
		Method          string
		Body            interface{}
		Headers         map[string]string
		QueryParams     map[string]string
		Timeout         time.Duration
		FollowRedirects bool
	}

	// Response is a struct that contains the data returned from an HTTP request.
	Response struct {
		Body       []byte
		Headers    map[string]string
		StatusCode int
		IsError    bool
		Empty      bool
		URL        string
	}

	// Client is a struct that contains the data needed to make an HTTP request.
	Client struct {
		Host           string
		DefaultTimeout time.Duration
		DefaultHeaders map[string]string
		Debug          bool
		httpClient     *http.Client
	}

	// ClientOptions contains options for configuring the HTTP client.
	ClientOptions struct {
		Timeout         time.Duration
		DefaultHeaders  map[string]string
		Debug           bool
		FollowRedirects bool
	}
)

// NewClient creates a new Client with default settings.
func NewClient(host string) *Client {
	return NewClientWithOptions(host, ClientOptions{
		Timeout:         DefaultTimeout,
		DefaultHeaders:  make(map[string]string, 0),
		Debug:           false,
		FollowRedirects: true,
	})
}

// NewClientWithOptions creates a new Client with custom options.
func NewClientWithOptions(host string, options ClientOptions) *Client {
	timeout := options.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	defaultHeaders := options.DefaultHeaders
	if defaultHeaders == nil {
		defaultHeaders = make(map[string]string)
	}

	return &Client{
		Host:           strings.TrimRight(host, "/"),
		DefaultTimeout: timeout,
		DefaultHeaders: defaultHeaders,
		Debug:          options.Debug,
		httpClient: &http.Client{
			Timeout:       timeout,
			CheckRedirect: defaultCheckRedirect(options.FollowRedirects),
		},
	}
}

// SendRequest makes an HTTP request.
// nolint:gocyclo
// nolint:cyclop
func (c *Client) SendRequest(data *Request) (*Response, error) {
	if data == nil {
		return nil, fmt.Errorf("request data cannot be nil")
	}

	if data.Method == "" {
		data.Method = http.MethodGet
	}

	timeout := data.Timeout
	if timeout == 0 {
		timeout = c.DefaultTimeout
	}

	// Build URL with query parameters
	baseURL := fmt.Sprintf("%s%s", c.Host, data.Path)
	if len(data.QueryParams) > 0 {
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}

		query := u.Query()
		for k, v := range data.QueryParams {
			query.Set(k, v)
		}
		u.RawQuery = query.Encode()
		baseURL = u.String()
	}

	var body io.Reader
	if data.Body != nil {
		b, err := createBody(data.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request body: %w", err)
		}
		body = b
	}

	if c.Debug {
		log.Printf("Making request: %s %s", data.Method, baseURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, data.Method, baseURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	headers := make(map[string]string)
	maps.Copy(headers, c.DefaultHeaders)
	maps.Copy(headers, data.Headers)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := c.httpClient
	if data.Timeout != 0 && data.Timeout != c.DefaultTimeout {
		client = &http.Client{
			Timeout:       data.Timeout,
			CheckRedirect: defaultCheckRedirect(data.FollowRedirects),
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && c.Debug {
			log.Printf("Warning: failed to close response body: %v", closeErr)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.Debug {
		log.Printf("Response: %s %s - Status: %d, Body length: %d",
			data.Method, baseURL, resp.StatusCode, len(respBody))
	}

	respHeaders := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	isError := resp.StatusCode >= 400
	empty := len(respBody) == 0

	return &Response{
		Body:       respBody,
		Headers:    respHeaders,
		StatusCode: resp.StatusCode,
		IsError:    isError,
		Empty:      empty,
		URL:        resp.Request.URL.String(),
	}, nil
}

// Get makes a GET request.
func (c *Client) Get(path string) (*Response, error) {
	return c.SendRequest(&Request{
		Path:   path,
		Method: http.MethodGet,
	})
}

// Post makes a POST request with a JSON body.
func (c *Client) Post(path string, body interface{}) (*Response, error) {
	return c.SendRequest(&Request{
		Path:   path,
		Method: http.MethodPost,
		Body:   body,
	})
}

// Put makes a PUT request with a JSON body.
func (c *Client) Put(path string, body interface{}) (*Response, error) {
	return c.SendRequest(&Request{
		Path:   path,
		Method: http.MethodPut,
		Body:   body,
	})
}

// Delete makes a DELETE request.
func (c *Client) Delete(path string) (*Response, error) {
	return c.SendRequest(&Request{
		Path:   path,
		Method: http.MethodDelete,
	})
}

// SetDefaultHeader sets a default header for all requests.
func (c *Client) SetDefaultHeader(key, value string) {
	if c.DefaultHeaders == nil {
		c.DefaultHeaders = make(map[string]string)
	}
	c.DefaultHeaders[key] = value
}

// IsJSON checks if the response is a JSON response.
func (rs *Response) IsJSON() bool {
	val, ok := rs.Headers["Content-Type"]
	if !ok {
		return false
	}
	return strings.Contains(val, "application/json")
}

func defaultCheckRedirect(followRedirects bool) func(req *http.Request, via []*http.Request) error {
	if !followRedirects {
		return func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return nil
}

func createBody(body interface{}) (io.Reader, error) {
	var reader io.Reader
	switch v := body.(type) {
	case string:
		reader = strings.NewReader(v)
	case []byte:
		reader = bytes.NewReader(v)
	case io.Reader:
		reader = v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	return reader, nil
}
