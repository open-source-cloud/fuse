package tests

import (
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/packages/debug"
	"github.com/open-source-cloud/fuse/internal/packages/logic"
)

// PackageRegistryWithInternalPackages returns a package registry with the internal packages
func PackageRegistryWithInternalPackages() packages.Registry {
	registry := packages.NewPackageRegistry()

	listOfInternalPackages := []packages.Package{
		debug.New(),
		logic.New(),
	}

	for _, pkg := range listOfInternalPackages {
		registry.Register(pkg.ID(), pkg)
	}

	return registry
}
