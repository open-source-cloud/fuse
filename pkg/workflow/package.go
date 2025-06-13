// Package packages workflow function packages
package workflow

// NewPackage creates a new Package
func NewPackage(id string, functions ...*PackagedFunction) *Package {
	return &Package{
		ID:        id,
		Functions: functions,
	}
}

// NewFunction creates a new packaged Function
func NewFunction(id string, metadata FunctionMetadata, fn Function) *PackagedFunction {
	return &PackagedFunction{
		ID:       id,
		Metadata: metadata,
		Function: fn,
	}
}

type (
	// Package workflow function Package
	Package struct {
		ID        string      `json:"id"`
		Functions []*PackagedFunction `json:"functions"`
	}

	// PackagedFunction packaged Function
	PackagedFunction struct {
		ID       string           `json:"id"`
		Metadata FunctionMetadata `json:"metadata"`
		Function Function
	}
)
