package http

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/http"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

// HTTPFunctionID is the id of the request function
const HTTPFunctionID = "request"

var (
	// ErrURLRequired is the error returned when the url is required
	ErrURLRequired = errors.New("url is required")
	// ErrMethodRequired is the error returned when the method is required
	ErrMethodRequired = errors.New("method is required")
	// ErrMethodNotAllowed is the error returned when the method is not allowed
	ErrMethodNotAllowed = errors.New("method not allowed")
)

// RequestFunctionMetadata returns the metadata for the request function
func RequestFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			CustomParameters: true,
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "host",
					Type:        "string",
					Required:    true,
					Validations: nil,
					Description: "The host to request",
					Default:     "",
				},
				{
					Name:        "path",
					Type:        "string",
					Required:    true,
					Validations: nil,
					Description: "The path to request",
					Default:     "",
				},
				{
					Name:        "method",
					Type:        "string",
					Required:    true,
					Validations: nil,
					Description: "The HTTP method to use",
					Default:     "GET",
				},
				{
					Name:        "body",
					Type:        "string",
					Required:    false,
					Validations: nil,
					Description: "The body of the request",
					Default:     "",
				},
				{
					Name:        "headers",
					Type:        "string",
					Required:    false,
					Validations: nil,
					Description: "The headers of the request",
					Default:     "",
				},
				{
					Name:        "timeout",
					Type:        "int",
					Required:    false,
					Validations: nil,
					Description: "The timeout of the request",
					Default:     10,
				},
			},
			Edges: workflow.InputEdgeMetadata{
				Count:      0,
				Parameters: make([]workflow.ParameterSchema, 0),
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "body",
					Type:        "[]byte",
					Required:    true,
					Validations: nil,
					Description: "The raw body of the response",
					Default:     []byte{},
				},
				{
					Name:        "status",
					Type:        "int",
					Required:    true,
					Validations: nil,
					Description: "The status of the response",
					Default:     200,
				},
				{
					Name:        "headers",
					Type:        "map[string]string",
					Required:    true,
					Validations: nil,
					Description: "The headers of the response",
					Default:     map[string]string{},
				},
				{
					Name:        "json",
					Type:        "map[string]any",
					Required:    false,
					Validations: nil,
					Description: "The JSON body of the response parsed as map[string]any, if the body is not valid JSON, the value will be nil",
					Default:     nil,
				},
			},
			Edges: make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// RequestFunction executes the request function
func RequestFunction(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	input := execInfo.Input

	request, err := makeRequestSchema(input)
	if err != nil {
		log.Err(err).Msgf("Error making request schema: %+v", request)
		return workflow.NewFunctionResult(workflow.FunctionError, map[string]any{"error": err.Error()}), err
	}

	host := input.GetStr("host")
	client := http.NewClient(host)

	response, err := client.SendRequest(request)
	if err != nil {
		log.Err(err).Msgf("Error making request: %+v", request)
		return workflow.NewFunctionResult(workflow.FunctionError, map[string]any{"error": err.Error()}), err
	}

	// if the response is a JSON response and the body is not empty, we need to unmarshal the body
	jsonBody := map[string]any{}
	if response.IsJSON() && len(response.Body) > 0 {
		err := json.Unmarshal(response.Body, &jsonBody)
		if err != nil {
			log.Err(err).Msgf("Error unmarshalling JSON body: %+v", response.Body)
		}
	}

	return workflow.NewFunctionResult(workflow.FunctionSuccess, map[string]any{
		"body":    response.Body,
		"status":  response.StatusCode,
		"headers": response.Headers,
		"json":    jsonBody,
	}), nil
}

// makeRequestSchema makes a request schema from the input
func makeRequestSchema(input *workflow.FunctionInput) (*http.Request, error) {
	request := &http.Request{}

	path := input.GetStr("path")
	if path == "" {
		return nil, ErrURLRequired
	}
	request.Path = path

	method := input.GetStr("method")
	if method == "" {
		return nil, ErrMethodRequired
	}
	request.Method = method

	body := input.GetStr("body")
	if body != "" {
		request.Body = body
	}

	headers := input.GetMapStr("headers")
	log.Debug().Msgf("Headers: %+v", headers)

	// if headers is not empty, we need to add it to the request
	if len(headers) > 0 {
		request.Headers = headers
	}

	timeout := input.GetInt("timeout")
	if timeout == 0 {
		timeout = 10
	}
	request.Timeout = time.Duration(timeout) * time.Second

	return request, nil
}
