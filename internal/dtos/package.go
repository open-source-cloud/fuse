package dtos

import (
	"github.com/open-source-cloud/fuse/pkg/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// RegisterPackageResponse represents package registration response
type RegisterPackageResponse struct {
	Message   string `json:"message" example:"Package registered successfully"`
	PackageID string `json:"packageId" example:"my-package"`
}

// PackageListResponse represents a paginated list of packages
type PackageListResponse struct {
	Metadata PaginationMetadata `json:"metadata"`
	Items    []PackageDTO       `json:"items"`
}

// PackageDTO represents a package data transfer object
type PackageDTO struct {
	ID        string                `json:"id" example:"my-package"`
	Functions []PackagedFunctionDTO `json:"functions"`
	Tags      map[string]string     `json:"tags,omitempty"`
}

// PackagedFunctionDTO represents a packaged function data transfer object
type PackagedFunctionDTO struct {
	ID       string              `json:"id" example:"my-function"`
	Metadata FunctionMetadataDTO `json:"metadata"`
	Tags     map[string]string   `json:"tags,omitempty"`
}

// FunctionMetadataDTO represents function metadata data transfer object
type FunctionMetadataDTO struct {
	Transport string            `json:"transport" example:"sync"`
	Input     InputMetadataDTO  `json:"input"`
	Output    OutputMetadataDTO `json:"output,omitempty"`
}

// InputMetadataDTO represents input metadata data transfer object
type InputMetadataDTO struct {
	CustomParameters bool                 `json:"customParameters"`
	Parameters       []ParameterSchemaDTO `json:"parameters"`
}

// ParameterSchemaDTO represents parameter schema data transfer object
type ParameterSchemaDTO struct {
	Name        string   `json:"name" example:"value"`
	Type        string   `json:"type" example:"string"`
	Description string   `json:"description" example:"Input value"`
	Required    bool     `json:"required" example:"true"`
	Validations []string `json:"validations,omitempty"`
	Default     any      `json:"default,omitempty"`
}

// OutputMetadataDTO represents output metadata data transfer object
type OutputMetadataDTO struct {
	Parameters             []ParameterSchemaDTO `json:"parameters"`
	ConditionalOutput      bool                 `json:"conditionalOutput" example:"false"`
	ConditionalOutputField string               `json:"conditionalOutputField,omitempty"`
}

// ToPackageDTO converts a workflow.Package to PackageDTO
func ToPackageDTO(pkg *workflow.Package) PackageDTO {
	if pkg == nil {
		return PackageDTO{}
	}

	functions := make([]PackagedFunctionDTO, len(pkg.Functions))
	for i, fn := range pkg.Functions {
		functions[i] = ToPackagedFunctionDTO(fn)
	}

	return PackageDTO{
		ID:        pkg.ID,
		Functions: functions,
		Tags:      pkg.Tags,
	}
}

// ToPackagedFunctionDTO converts a workflow.PackagedFunction to PackagedFunctionDTO
func ToPackagedFunctionDTO(fn *workflow.PackagedFunction) PackagedFunctionDTO {
	if fn == nil {
		return PackagedFunctionDTO{}
	}

	return PackagedFunctionDTO{
		ID:       fn.ID,
		Metadata: ToFunctionMetadataDTO(fn.Metadata),
		Tags:     fn.Tags,
	}
}

// ToFunctionMetadataDTO converts workflow.FunctionMetadata to FunctionMetadataDTO
func ToFunctionMetadataDTO(meta workflow.FunctionMetadata) FunctionMetadataDTO {
	inputParams := make([]ParameterSchemaDTO, len(meta.Input.Parameters))
	for i, p := range meta.Input.Parameters {
		inputParams[i] = ParameterSchemaDTO{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Required:    p.Required,
			Validations: p.Validations,
			Default:     p.Default,
		}
	}

	outputParams := make([]ParameterSchemaDTO, len(meta.Output.Parameters))
	for i, p := range meta.Output.Parameters {
		outputParams[i] = ParameterSchemaDTO{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Required:    p.Required,
			Validations: p.Validations,
			Default:     p.Default,
		}
	}

	return FunctionMetadataDTO{
		Transport: string(meta.Transport),
		Input: InputMetadataDTO{
			CustomParameters: meta.Input.CustomParameters,
			Parameters:       inputParams,
		},
		Output: OutputMetadataDTO{
			Parameters:             outputParams,
			ConditionalOutput:      meta.Output.ConditionalOutput,
			ConditionalOutputField: meta.Output.ConditionalOutputField,
		},
	}
}

// FromPackageDTO converts a PackageDTO to workflow.Package
func FromPackageDTO(dto PackageDTO) *workflow.Package {
	functions := make([]*workflow.PackagedFunction, len(dto.Functions))
	for i, fn := range dto.Functions {
		functions[i] = FromPackagedFunctionDTO(fn)
	}

	return &workflow.Package{
		ID:        dto.ID,
		Functions: functions,
		Tags:      dto.Tags,
	}
}

// FromPackagedFunctionDTO converts a PackagedFunctionDTO to workflow.PackagedFunction
func FromPackagedFunctionDTO(dto PackagedFunctionDTO) *workflow.PackagedFunction {
	inputParams := make([]workflow.ParameterSchema, len(dto.Metadata.Input.Parameters))
	for i, p := range dto.Metadata.Input.Parameters {
		inputParams[i] = workflow.ParameterSchema{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Required:    p.Required,
			Validations: p.Validations,
			Default:     p.Default,
		}
	}

	outputParams := make([]workflow.ParameterSchema, len(dto.Metadata.Output.Parameters))
	for i, p := range dto.Metadata.Output.Parameters {
		outputParams[i] = workflow.ParameterSchema{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Required:    p.Required,
			Validations: p.Validations,
			Default:     p.Default,
		}
	}

	return &workflow.PackagedFunction{
		ID: dto.ID,
		Metadata: workflow.FunctionMetadata{
			Transport: transport.Type(dto.Metadata.Transport),
			Input: workflow.InputMetadata{
				CustomParameters: dto.Metadata.Input.CustomParameters,
				Parameters:       inputParams,
			},
			Output: workflow.OutputMetadata{
				Parameters:             outputParams,
				ConditionalOutput:      dto.Metadata.Output.ConditionalOutput,
				ConditionalOutputField: dto.Metadata.Output.ConditionalOutputField,
			},
		},
		Tags: dto.Tags,
	}
}
