package dtos

import "github.com/open-source-cloud/fuse/pkg/workflow"

// CredentialDTO is the read shape of a credential. It carries metadata only — field values are
// never returned (ADR-0031).
type CredentialDTO struct {
	ID          string   `json:"id" example:"openai-prod"`
	Type        string   `json:"type" example:"openai"`
	Description string   `json:"description,omitempty" example:"Production OpenAI credentials"`
	Fields      []string `json:"fields"`
}

// UpsertCredentialRequest is the write shape. Field values are write-only and are stored in the
// SecretStore for the target environment; they never appear in a read response.
type UpsertCredentialRequest struct {
	Type        string            `json:"type" example:"openai"`
	Description string            `json:"description,omitempty"`
	Fields      map[string]string `json:"fields"`
}

// CredentialListResponse represents a list of credentials (metadata only).
type CredentialListResponse struct {
	Items []CredentialDTO `json:"items"`
}

// UpsertCredentialResponse represents a credential upsert response.
type UpsertCredentialResponse struct {
	Message    string `json:"message" example:"Credential saved successfully"`
	Credential string `json:"credential" example:"openai-prod"`
}

// ToCredentialDTO converts credential metadata to its read DTO (never includes values).
func ToCredentialDTO(c *workflow.Credential) CredentialDTO {
	return CredentialDTO{ID: c.ID, Type: c.Type, Description: c.Description, Fields: c.Fields}
}
