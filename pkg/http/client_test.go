package http_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpClient "github.com/open-source-cloud/fuse/pkg/http"
	"github.com/stretchr/testify/suite"
)

type HTTPClientTestSuite struct {
	suite.Suite
	server *httptest.Server
	client *httpClient.Client
}

func TestHTTPClientSuite(t *testing.T) {
	suite.Run(t, new(HTTPClientTestSuite))
}

func (s *HTTPClientTestSuite) SetupTest() {
	// Create a test server for each test
	s.server = httptest.NewServer(http.HandlerFunc(s.testHandler))
	s.client = httpClient.NewClient(s.server.URL)
}

func (s *HTTPClientTestSuite) TearDownTest() {
	if s.server != nil {
		s.server.Close()
	}
}

// testHandler handles different test scenarios based on the request path
func (s *HTTPClientTestSuite) testHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/success":
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"message": "success", "method": "%s"}`, r.Method)

	case "/echo":
		// Echo back request details
		body, _ := io.ReadAll(r.Body)
		response := map[string]interface{}{
			"method":  r.Method,
			"headers": r.Header,
			"query":   r.URL.Query(),
			"body":    string(body),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)

	case "/error":
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "Internal Server Error")

	case "/timeout":
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)

	case "/redirect":
		http.Redirect(w, r, "/success", http.StatusMovedPermanently)

	case "/empty":
		w.WriteHeader(http.StatusOK)

	case "/json":
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"response": "json"})

	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, "Not Found")
	}
}

// Test client creation
func (s *HTTPClientTestSuite) TestNewClient() {
	client := httpClient.NewClient("https://example.com")
	s.NotNil(client)
	s.Equal("https://example.com", client.Host)
	s.Equal(30*time.Second, client.DefaultTimeout)
	s.False(client.Debug)
	s.NotNil(client.DefaultHeaders)
}

func (s *HTTPClientTestSuite) TestNewClientWithOptions() {
	options := httpClient.ClientOptions{
		Timeout:         10 * time.Second,
		DefaultHeaders:  map[string]string{"User-Agent": "test-agent"},
		Debug:           true,
		FollowRedirects: false,
	}

	client := httpClient.NewClientWithOptions("https://example.com", options)
	s.NotNil(client)
	s.Equal("https://example.com", client.Host)
	s.Equal(10*time.Second, client.DefaultTimeout)
	s.True(client.Debug)
	s.Equal("test-agent", client.DefaultHeaders["User-Agent"])
}

func (s *HTTPClientTestSuite) TestNewClientWithOptionsDefaults() {
	// Test with empty options to ensure defaults are applied
	options := httpClient.ClientOptions{}
	client := httpClient.NewClientWithOptions("https://example.com", options)
	s.Equal(30*time.Second, client.DefaultTimeout)
	s.NotNil(client.DefaultHeaders)
	s.False(client.Debug)
}

// Test basic request functionality
func (s *HTTPClientTestSuite) TestRequestSuccess() {
	req := &httpClient.Request{
		Path:   "/success",
		Method: "GET",
	}

	resp, err := s.client.Request(req)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.False(resp.IsError)
	s.False(resp.Empty)
	s.Contains(string(resp.Body), "success")
}

func (s *HTTPClientTestSuite) TestRequestNilData() {
	resp, err := s.client.Request(nil)
	s.Error(err)
	s.Nil(resp)
	s.Contains(err.Error(), "request data cannot be nil")
}

func (s *HTTPClientTestSuite) TestRequestDefaultMethod() {
	req := &httpClient.Request{
		Path: "/echo",
		// Method is empty, should default to GET
	}

	resp, err := s.client.Request(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)
	s.Equal("GET", responseData["method"])
}

// Test different HTTP methods
func (s *HTTPClientTestSuite) TestRequestMethods() {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		s.Run(method, func() {
			req := &httpClient.Request{
				Path:   "/echo",
				Method: method,
			}

			resp, err := s.client.Request(req)
			s.NoError(err)
			s.Equal(http.StatusOK, resp.StatusCode)

			var responseData map[string]interface{}
			err = json.Unmarshal(resp.Body, &responseData)
			s.NoError(err)
			s.Equal(method, responseData["method"])
		})
	}
}

// Test request body handling
func (s *HTTPClientTestSuite) TestRequestBodyString() {
	req := &httpClient.Request{
		Path:   "/echo",
		Method: "POST",
		Body:   "test string body",
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)
	s.Equal("test string body", responseData["body"])
}

func (s *HTTPClientTestSuite) TestRequestBodyBytes() {
	bodyBytes := []byte("test bytes body")
	req := &httpClient.Request{
		Path:   "/echo",
		Method: "POST",
		Body:   bodyBytes,
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)
	s.Equal("test bytes body", responseData["body"])
}

func (s *HTTPClientTestSuite) TestRequestBodyReader() {
	bodyReader := strings.NewReader("test reader body")
	req := &httpClient.Request{
		Path:   "/echo",
		Method: "POST",
		Body:   bodyReader,
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)
	s.Equal("test reader body", responseData["body"])
}

func (s *HTTPClientTestSuite) TestRequestBodyJSON() {
	bodyData := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	req := &httpClient.Request{
		Path:   "/echo",
		Method: "POST",
		Body:   bodyData,
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)

	var bodyJSON map[string]interface{}
	err = json.Unmarshal([]byte(responseData["body"].(string)), &bodyJSON)
	s.NoError(err)
	s.Equal("value1", bodyJSON["key1"])
	s.Equal(float64(123), bodyJSON["key2"])
}

// Test headers
func (s *HTTPClientTestSuite) TestRequestHeaders() {
	req := &httpClient.Request{
		Path:   "/echo",
		Method: "GET",
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
			"Authorization":   "Bearer token123",
		},
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)

	headers := responseData["headers"].(map[string]interface{})
	s.Contains(headers, "X-Custom-Header")
	s.Contains(headers, "Authorization")
}

func (s *HTTPClientTestSuite) TestDefaultHeaders() {
	s.client.SetDefaultHeader("X-Default-Header", "default-value")

	req := &httpClient.Request{
		Path:   "/echo",
		Method: "GET",
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)

	headers := responseData["headers"].(map[string]interface{})
	s.Contains(headers, "X-Default-Header")
}

func (s *HTTPClientTestSuite) TestHeaderOverride() {
	s.client.SetDefaultHeader("X-Test-Header", "default-value")

	req := &httpClient.Request{
		Path:   "/echo",
		Method: "GET",
		Headers: map[string]string{
			"X-Test-Header": "override-value",
		},
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)

	headers := responseData["headers"].(map[string]interface{})
	headerValues := headers["X-Test-Header"].([]interface{})
	s.Equal("override-value", headerValues[0])
}

// Test query parameters
func (s *HTTPClientTestSuite) TestQueryParameters() {
	req := &httpClient.Request{
		Path:   "/echo",
		Method: "GET",
		QueryParams: map[string]string{
			"param1": "value1",
			"param2": "value2",
		},
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)

	query := responseData["query"].(map[string]interface{})
	s.Contains(query, "param1")
	s.Contains(query, "param2")
}

// Test convenience methods
func (s *HTTPClientTestSuite) TestGet() {
	resp, err := s.client.Get("/success")
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *HTTPClientTestSuite) TestPost() {
	bodyData := map[string]string{"key": "value"}
	resp, err := s.client.Post("/echo", bodyData)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)
	s.Equal("POST", responseData["method"])
}

func (s *HTTPClientTestSuite) TestPut() {
	bodyData := map[string]string{"key": "value"}
	resp, err := s.client.Put("/echo", bodyData)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)
	s.Equal("PUT", responseData["method"])
}

func (s *HTTPClientTestSuite) TestDelete() {
	resp, err := s.client.Delete("/echo")
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)
	s.Equal("DELETE", responseData["method"])
}

// Test error handling
func (s *HTTPClientTestSuite) TestHTTPError() {
	resp, err := s.client.Get("/error")
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, resp.StatusCode)
	s.True(resp.IsError)
	s.Contains(string(resp.Body), "Internal Server Error")
}

func (s *HTTPClientTestSuite) TestTimeout() {
	req := &httpClient.Request{
		Path:    "/timeout",
		Method:  "GET",
		Timeout: 500 * time.Millisecond,
	}

	resp, err := s.client.Request(req)
	s.Error(err)
	s.Nil(resp)
	s.Contains(err.Error(), "request failed")
}

func (s *HTTPClientTestSuite) TestInvalidURL() {
	// Create a client with an invalid host URL that would cause url.Parse to fail
	invalidClient := httpClient.NewClient("ht!tp://invalid url with spaces")

	req := &httpClient.Request{
		Path:   "/test",
		Method: "GET",
	}

	resp, err := invalidClient.Request(req)
	s.Error(err)
	s.Nil(resp)
}

// Test response handling
func (s *HTTPClientTestSuite) TestEmptyResponse() {
	resp, err := s.client.Get("/empty")
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.True(resp.Empty)
	s.Empty(resp.Body)
}

// Test redirect handling
func (s *HTTPClientTestSuite) TestRedirectFollowed() {
	resp, err := s.client.Get("/redirect")
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(string(resp.Body), "success")
}

func (s *HTTPClientTestSuite) TestRedirectNotFollowed() {
	options := httpClient.ClientOptions{
		FollowRedirects: false,
	}
	client := httpClient.NewClientWithOptions(s.server.URL, options)

	resp, err := client.Get("/redirect")
	s.NoError(err)
	s.Equal(http.StatusMovedPermanently, resp.StatusCode)
}

// Test debug mode
func (s *HTTPClientTestSuite) TestDebugMode() {
	s.client.Debug = true

	req := &httpClient.Request{
		Path:   "/success",
		Method: "GET",
		Debug:  true,
	}

	// This mainly tests that debug logging doesn't cause errors
	resp, err := s.client.Request(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
}

// Test custom timeout per request
func (s *HTTPClientTestSuite) TestCustomTimeoutPerRequest() {
	req := &httpClient.Request{
		Path:    "/success",
		Method:  "GET",
		Timeout: 5 * time.Second,
	}

	resp, err := s.client.Request(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
}

// Test host trimming
func (s *HTTPClientTestSuite) TestHostTrimming() {
	client := httpClient.NewClient(s.server.URL + "/")
	s.Equal(s.server.URL, client.Host)
}

// Test automatic Content-Type setting for JSON
func (s *HTTPClientTestSuite) TestAutoContentTypeJSON() {
	req := &httpClient.Request{
		Path:   "/echo",
		Method: "POST",
		Body:   map[string]string{"key": "value"},
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)

	headers := responseData["headers"].(map[string]interface{})
	contentType := headers["Content-Type"].([]interface{})
	s.Equal("application/json", contentType[0])
}

// Test that string/bytes bodies don't get automatic Content-Type
func (s *HTTPClientTestSuite) TestNoAutoContentTypeForStringBytes() {
	req := &httpClient.Request{
		Path:   "/echo",
		Method: "POST",
		Body:   "string body",
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)

	headers := responseData["headers"].(map[string]interface{})
	// Should not have automatic Content-Type for string bodies
	if contentType, exists := headers["Content-Type"]; exists {
		s.NotEqual("application/json", contentType.([]interface{})[0])
	}
}

// Test Content-Type override
func (s *HTTPClientTestSuite) TestContentTypeOverride() {
	req := &httpClient.Request{
		Path:   "/echo",
		Method: "POST",
		Body:   map[string]string{"key": "value"},
		Headers: map[string]string{
			"Content-Type": "application/xml",
		},
	}

	resp, err := s.client.Request(req)
	s.NoError(err)

	var responseData map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseData)
	s.NoError(err)

	headers := responseData["headers"].(map[string]interface{})
	contentType := headers["Content-Type"].([]interface{})
	s.Equal("application/xml", contentType[0])
}

// Benchmark tests
func BenchmarkHTTPClientGet(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status": "ok"}`)
	}))
	defer server.Close()

	client := httpClient.NewClient(server.URL)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Get("/test")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTPClientPost(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status": "ok"}`)
	}))
	defer server.Close()

	client := httpClient.NewClient(server.URL)
	body := map[string]string{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Post("/test", body)
		if err != nil {
			b.Fatal(err)
		}
	}
}
