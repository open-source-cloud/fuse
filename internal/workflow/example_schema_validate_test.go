package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validateExampleWorkflowJSONFile(t *testing.T, dir, name string) {
	t.Helper()
	t.Parallel()
	//nolint:gosec // G304: path is only files returned by os.ReadDir from examples/workflows
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	schema, err := NewGraphSchemaFromJSON(raw)
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	if err := schema.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if _, err := NewGraph(schema); err != nil {
		t.Fatalf("NewGraph: %v", err)
	}
}

func TestExampleWorkflowJSONSchemasValidate(t *testing.T) {
	t.Parallel()
	dir := filepath.Join("..", "..", "examples", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			validateExampleWorkflowJSONFile(t, dir, name)
		})
	}
}
