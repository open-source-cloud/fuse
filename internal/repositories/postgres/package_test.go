package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageObjectKey(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"system", "packages/system/definition.json"},
		{"fuse/pkg/debug", "packages/debug/definition.json"},
		{"fuse/pkg/http", "packages/http/definition.json"},
		{"fuse/pkg/logic", "packages/logic/definition.json"},
		{"acme/foo", "packages/acme/foo/definition.json"},
		{"fuse/pkg", "packages/fuse/pkg/definition.json"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			assert.Equal(t, tt.want, packageObjectKey(tt.id))
		})
	}
}
