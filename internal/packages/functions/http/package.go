package http

import "github.com/open-source-cloud/fuse/pkg/workflow"

const PackageID = "fuse/pkg/http"

type HttpPackage struct{}

func New() *workflow.Package {
	return workflow.NewPackage(
		PackageID,
		workflow.NewFunction(
			HTTPFunctionID,
			RequestFunctionMetadata(),
			RequestFunction,
		),
	)
}
