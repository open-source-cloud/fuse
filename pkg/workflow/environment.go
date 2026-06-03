package workflow

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// DefaultEnvironmentName is the environment used when a trigger does not specify one. It is
// always valid and seeded into the environments registry (ADR-0031).
const DefaultEnvironmentName = "default"

// environmentNamePattern restricts environment names to lowercase alphanumerics plus -._ so they
// are safe as secret-store scope keys and URL path segments.
var environmentNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.\-]*$`)

// Environment is a declared resolution scope (e.g. dev/staging/prod) referenced by workflow
// executions to resolve secrets (ADR-0031).
type Environment struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
}

// NewEnvironment creates an Environment.
func NewEnvironment(name, description string) *Environment {
	return &Environment{Name: name, Description: description}
}

// Validate checks the environment's fields and name format.
func (e *Environment) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(e); err != nil {
		return err
	}
	if !environmentNamePattern.MatchString(e.Name) {
		return fmt.Errorf("invalid environment name %q: must match %s", e.Name, environmentNamePattern.String())
	}
	return nil
}
