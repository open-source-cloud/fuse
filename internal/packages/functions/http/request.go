package http

import (
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/open-source-cloud/fuse/pkg/http"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

const HTTPFunctionID = "request"

var (
	ErrURLRequired      = errors.New("url is required")
	ErrMethodRequired   = errors.New("method is required")
	ErrMethodNotAllowed = errors.New("method not allowed")
)

func RequestFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
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
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "body",
					Type:        "[]byte",
					Required:    true,
					Validations: nil,
					Description: "The body of the response",
					Default:     "",
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
					Default:     "",
				},
			},
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

	response, err := client.Request(request)
	if err != nil {
		log.Err(err).Msgf("Error making request: %+v", request)
		return workflow.NewFunctionResult(workflow.FunctionError, map[string]any{"error": err.Error()}), err
	}

	log.Info().Msg("Request made successfully")
	log.Debug().Msgf("Response: %+v", response)

	return workflow.NewFunctionResult(workflow.FunctionSuccess, makeResponseSchema(response)), nil
}

// makeRequestSchema makes a request schema from the input
func makeRequestSchema(input *workflow.FunctionInput) (http.Request, error) {
	// TODO: Think about binding validations like schema or use a library like go-jsonschema
	// TODO: Add support for query params

	request := http.Request{}

	path := input.GetStr("path")
	if path == "" {
		return request, ErrURLRequired
	}
	request.Path = path

	method := input.GetStr("method")
	if method == "" {
		return request, ErrMethodRequired
	}
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	if !slices.Contains(allowedMethods, method) {
		return request, ErrMethodNotAllowed
	}
	request.Method = method

	body := input.GetStr("body")
	if body != "" {
		request.Body = body
	}

	headers := input.GetMap("headers")
	// if headers is not empty, we need to add it to the request
	if len(headers) > 0 {
		if request.Headers == nil {
			request.Headers = make(map[string]string)
		}
		maps.Copy(request.Headers, headers)
	}

	timeout := input.GetInt("timeout")
	if timeout == 0 {
		timeout = 10
	}
	request.Timeout = time.Duration(timeout) * time.Second

	return request, nil
}

// makeResponseSchema makes a response schema from the input
func makeResponseSchema(response http.Response) map[string]any {
	return map[string]any{
		"body":    response.Body,
		"status":  response.StatusCode,
		"headers": response.Headers,
	}
}
