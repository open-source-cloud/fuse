package packages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/open-source-cloud/fuse/app/config"
	workflow "github.com/open-source-cloud/fuse/pkg/workflow"
)

type internalFunction struct {
	id       string
	metadata workflow.FunctionMetadata
	fn       workflow.Function
}

// NewInternalFunction creates a new internal function
func NewInternalFunction(packageID string, id string, metadata workflow.FunctionMetadata, fn workflow.Function) FunctionSpec {
	return &internalFunction{
		id:       fmt.Sprintf("%s/%s", packageID, id),
		metadata: metadata,
		fn:       fn,
	}
}

func (f *internalFunction) ID() string {
	return f.id
}

func (f *internalFunction) Metadata() workflow.FunctionMetadata {
	return f.metadata
}

func (f *internalFunction) Execute(workflowID string, execID string, input *workflow.FunctionInput) (workflow.FunctionResult, error) {
	return f.fn(&workflow.ExecutionInfo{
		WorkflowID: workflowID,
		ExecID:     execID,
		Finish: func(result workflow.FunctionOutput) {
			port := config.Instance().Server.Port
			url := fmt.Sprintf("http://localhost:%d/v1/workflows/%s/execs/%s", port, workflowID, execID)

			payload := map[string]any{"execID": execID, "result": result}
			jsonPayload, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, _ := client.Do(req)
			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(resp.Body)
		},
	}, input)
}
