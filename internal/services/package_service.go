package services

import (
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// PackageOptions represents the options for the package service
	PackageOptions struct {
		Load bool
	}
	// PackageService represents the transactional and logical service to manage workflow.Package
	PackageService interface {
		FindAll(opts PackageOptions) ([]*workflow.Package, error)
		FindByID(id string, opts PackageOptions) (*workflow.Package, error)
		Save(pkg *workflow.Package) (*workflow.Package, error)
		RegisterInternalPackages() error
	}
	// DefaultPackageService is the default implementation of the PackageService interface
	DefaultPackageService struct {
		PackageService
		packageRepo      repositories.PackageRepository
		packageRegistry  packages.Registry
		internalPackages packages.InternalPackages
	}
)

// NewPackageService returns a new PackageService
func NewPackageService(packageRepo repositories.PackageRepository, packageRegistry packages.Registry, internalPackages packages.InternalPackages) PackageService {
	return &DefaultPackageService{
		packageRepo:      packageRepo,
		packageRegistry:  packageRegistry,
		internalPackages: internalPackages,
	}
}

// FindAll finds all packages
func (s *DefaultPackageService) FindAll(opts PackageOptions) ([]*workflow.Package, error) {
	pkgs, err := s.packageRepo.FindAll()
	if err != nil {
		return nil, err
	}

	// Register the packages if they are not already registered
	// Eager load the packages to avoid multiple calls to the repository
	if opts.Load {
		for _, pkg := range pkgs {
			if !s.packageRegistry.Has(pkg.ID) {
				s.packageRegistry.Register(pkg)
			}
		}
	}

	return pkgs, nil
}

// FindByID finds a package by ID
func (s *DefaultPackageService) FindByID(id string, opts PackageOptions) (*workflow.Package, error) {
	pkg, err := s.packageRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Register the package if it is not already registered
	// Eager load the package to avoid multiple calls to the repository
	if opts.Load && !s.packageRegistry.Has(id) {
		s.packageRegistry.Register(pkg)
	}

	return pkg, nil
}

// Save saves a package to the repository and registry if it is not already registered
func (s *DefaultPackageService) Save(pkg *workflow.Package) (*workflow.Package, error) {
	if err := pkg.Validate(); err != nil {
		return nil, err
	}

	if err := s.packageRepo.Save(pkg); err != nil {
		return nil, err
	}

	s.packageRegistry.Register(pkg)

	return pkg, nil
}

// RegisterInternalPackages registers the internal packages to the repository and registry
func (s *DefaultPackageService) RegisterInternalPackages() error {
	wg := sync.WaitGroup{}
	wg.Add(len(s.internalPackages.List()))

	for _, pkg := range s.internalPackages.List() {
		go func(pkg *workflow.Package) {
			defer wg.Done()
			if _, err := s.Save(pkg); err != nil {
				log.Error().Err(err).Msgf("failed to save internal package %s", pkg.ID)
			}
		}(pkg)
	}

	wg.Wait()

	return nil
}
