package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertDSNForMigrate(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "postgres scheme is rewritten to pgx5",
			dsn:  "postgres://fuse:fuse@localhost:5432/fuse?sslmode=disable",
			want: "pgx5://fuse:fuse@localhost:5432/fuse?sslmode=disable",
		},
		{
			name: "postgresql scheme is rewritten to pgx5",
			dsn:  "postgresql://user:pw@ep-x-pooler.us-east-2.pg.laravel.cloud/main?sslmode=require",
			want: "pgx5://user:pw@ep-x-pooler.us-east-2.pg.laravel.cloud/main?sslmode=require",
		},
		{
			name: "already pgx5 is left untouched",
			dsn:  "pgx5://fuse:fuse@localhost:5432/fuse",
			want: "pgx5://fuse:fuse@localhost:5432/fuse",
		},
		{
			name: "keyword value DSN is left untouched",
			dsn:  "host=localhost port=5432 user=fuse dbname=fuse",
			want: "host=localhost port=5432 user=fuse dbname=fuse",
		},
		{
			name: "empty DSN is left untouched",
			dsn:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertDSNForMigrate(tt.dsn)

			assert.Equal(t, tt.want, got)
		})
	}
}
