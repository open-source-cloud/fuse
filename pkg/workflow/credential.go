package workflow

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// credentialIDPattern restricts credential ids to lowercase alphanumerics plus -._ so they are
// safe URL path segments. The absence of "/" keeps the reserved cred/<id>/<field> secret name
// from colliding with the {{secret:NAME}} namespace (ADR-0031).
var credentialIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.\-]*$`)

// credentialFieldPattern restricts field names to mixed-case alphanumerics plus -_ (no dot, which
// separates id from field in the {{credential:id.field}} token). camelCase like "apiKey" is allowed.
var credentialFieldPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_\-]*$`)

// Credential is typed, centrally-managed credential metadata (ADR-0031 Option B). Field VALUES are
// NOT stored here: they live in the SecretStore at cred/<id>/<field>, per environment. This struct
// only carries the field NAMES so the value plaintext never flows through metadata sinks.
type Credential struct {
	ID          string   `json:"id" validate:"required"`
	Type        string   `json:"type" validate:"required"`
	Description string   `json:"description,omitempty"`
	Fields      []string `json:"fields"`
}

// NewCredential creates a Credential from metadata (field names only).
func NewCredential(id, credType, description string, fields []string) *Credential {
	return &Credential{ID: id, Type: credType, Description: description, Fields: fields}
}

// Validate checks the credential's required fields and the id / field-name format.
func (c *Credential) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(c); err != nil {
		return err
	}
	if !credentialIDPattern.MatchString(c.ID) {
		return fmt.Errorf("invalid credential id %q: must match %s", c.ID, credentialIDPattern.String())
	}
	for _, f := range c.Fields {
		if !credentialFieldPattern.MatchString(f) {
			return fmt.Errorf("invalid credential field name %q: must match %s", f, credentialFieldPattern.String())
		}
	}
	return nil
}
