package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type (
	// Request is a struct that contains the data needed to make an HTTP request.
	Request struct {
		Path    string
		Method  string
		Body    interface{}
		Headers map[string]string
		Timeout time.Duration
		Debug   bool
	}

	// Response is a struct that contains the data returned from an HTTP request.
	Response struct {
		Body        []byte
		Headers     map[string]string
		StatusCode  int
		IsHttpError bool
		Empty       bool
	}
	// Client is a struct that contains the data needed to make an HTTP request.
	Client struct {
		Host string
	}
)

// NewClient creates a new Client.
func NewClient(host string) Client {
	return Client{
		Host: host,
	}
}

// Request makes an HTTP request.
func (c *Client) Request(data Request) (Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), data.Timeout)
	defer cancel()

	url := fmt.Sprintf("%s%s", c.Host, data.Path)

	var body = strings.NewReader("")
	if data.Body != nil {
		b, err := json.Marshal(data.Body)
		if err != nil {
			return Response{}, err
		}
		body = strings.NewReader(string(b))
	}

	if data.Debug {
		log.Printf("Making request to %s - %s", data.Method, url)
	}

	req, err := http.NewRequestWithContext(ctx, data.Method, url, body)
	if err != nil {
		return Response{}, err
	}

	for k, v := range data.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Response{}, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	if data.Debug {
		log.Printf("Response from %s - %s with status=%d and body=%s", data.Method, url, resp.StatusCode, string(respBody))
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Print("Failed to close response body")
		}
	}(resp.Body)

	respHeaders := map[string]string{}

	for k, v := range resp.Header {
		respHeaders[k] = v[0]
	}

	httpError := resp.StatusCode >= 400
	empty := len(respBody) == 0

	return Response{
		Body:        respBody,
		Headers:     respHeaders,
		StatusCode:  resp.StatusCode,
		IsHttpError: httpError,
		Empty:       empty,
	}, nil
}
