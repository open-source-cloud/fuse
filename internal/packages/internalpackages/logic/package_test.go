package logic_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/logic"
)

func TestPackage(t *testing.T) {
	pkg := logic.New()

	if pkg.ID() != logic.PackageID {
		t.Fatalf("logic package should have id logic")
	}

	if len(pkg.Functions()) == 0 {
		t.Fatalf("logic package should have at least one function")
	}
}
