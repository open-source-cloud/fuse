package repositories

import (
	"errors"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

var (
	// ErrPackageNotFound is returned when a package is not found
	ErrPackageNotFound = errors.New("package not found")
	// ErrPackageNotModified is returned when a package is not modified
	ErrPackageNotModified = errors.New("package not modified")
)

type (
	// PackageRepository defines the interface of a package repository
	PackageRepository interface {
		FindByID(id string) (*workflow.Package, error)
		FindAll() ([]*workflow.Package, error)
		Save(pkg *workflow.Package) error
		Delete(id string) error
	}
)
