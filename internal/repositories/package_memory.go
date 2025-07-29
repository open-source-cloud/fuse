package repositories

import "github.com/open-source-cloud/fuse/pkg/workflow"

// MemoryPackageRepository is a memory package repository
type MemoryPackageRepository struct {
	packages map[string]*workflow.Package
}

// NewMemoryPackageRepository creates a new memory package repository
func NewMemoryPackageRepository() *MemoryPackageRepository {
	return &MemoryPackageRepository{
		packages: make(map[string]*workflow.Package),
	}
}

// FindByID finds a package by ID in the memory repository
func (r *MemoryPackageRepository) FindByID(id string) (*workflow.Package, error) {
	pkg, ok := r.packages[id]
	if !ok {
		return nil, ErrPackageNotFound
	}
	return pkg, nil
}

// FindAll finds all packages in the memory repository
func (r *MemoryPackageRepository) FindAll() ([]*workflow.Package, error) {
	pkgs := make([]*workflow.Package, 0, len(r.packages))
	for _, pkg := range r.packages {
		pkgs = append(pkgs, pkg)
	}

	// if no packages are found, return an empty slice
	if len(pkgs) == 0 {
		return []*workflow.Package{}, nil
	}

	return pkgs, nil
}

// Save saves a package to the memory repository
func (r *MemoryPackageRepository) Save(pkg *workflow.Package) error {
	r.packages[pkg.ID] = pkg
	return nil
}

// Delete deletes a package from the memory repository
func (r *MemoryPackageRepository) Delete(id string) error {
	delete(r.packages, id)
	return nil
}
