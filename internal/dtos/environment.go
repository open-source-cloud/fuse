package dtos

import "github.com/open-source-cloud/fuse/pkg/workflow"

// EnvironmentDTO represents an environment data transfer object.
type EnvironmentDTO struct {
	Name        string `json:"name" example:"staging"`
	Description string `json:"description,omitempty" example:"Staging environment"`
}

// EnvironmentListResponse represents a list of environments.
type EnvironmentListResponse struct {
	Items []EnvironmentDTO `json:"items"`
}

// UpsertEnvironmentResponse represents an environment upsert response.
type UpsertEnvironmentResponse struct {
	Message     string `json:"message" example:"Environment saved successfully"`
	Environment string `json:"environment" example:"staging"`
}

// ToEnvironmentDTO converts a domain environment to its DTO.
func ToEnvironmentDTO(env *workflow.Environment) EnvironmentDTO {
	return EnvironmentDTO{Name: env.Name, Description: env.Description}
}

// FromEnvironmentDTO converts a DTO to a domain environment.
func FromEnvironmentDTO(dto EnvironmentDTO) *workflow.Environment {
	return workflow.NewEnvironment(dto.Name, dto.Description)
}
