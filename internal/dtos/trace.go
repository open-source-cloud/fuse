package dtos

import "github.com/open-source-cloud/fuse/internal/workflow"

// TraceListResponse represents a paginated list of execution traces
type TraceListResponse struct {
	Traces []*workflow.ExecutionTrace `json:"traces"`
	Total  int                        `json:"total"`
	Limit  int                        `json:"limit"`
	Offset int                        `json:"offset"`
}
