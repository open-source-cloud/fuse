package dtos

import "github.com/open-source-cloud/fuse/pkg/workflow"

type (
	// CreatePackageRequest is the request body for creating a new package
	CreatePackageRequest struct {
		Package *workflow.Package `json:"package" validate:"required,dive"`
	}
	// PackageCreatedResponse is the response body for creating a new package
	PackageCreatedResponse struct {
		Message   string `json:"message"`
		PackageID string `json:"packageId"`
	}
)
