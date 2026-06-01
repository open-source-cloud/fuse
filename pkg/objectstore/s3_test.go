package objectstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeS3Endpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		endpoint  string
		useSSL    bool
		wantHost  string
		wantSSL   bool
		wantError string
	}{
		{
			name:     "host port without scheme",
			endpoint: "localhost:9000",
			useSSL:   false,
			wantHost: "localhost:9000",
			wantSSL:  false,
		},
		{
			name:     "http URL strips scheme",
			endpoint: "http://localhost:4566",
			useSSL:   true,
			wantHost: "localhost:4566",
			wantSSL:  false,
		},
		{
			name:     "https URL enables TLS",
			endpoint: "https://s3.amazonaws.com",
			useSSL:   false,
			wantHost: "s3.amazonaws.com",
			wantSSL:  true,
		},
		{
			name:      "path in URL is rejected",
			endpoint:  "http://localhost:9000/bucket",
			wantError: "must not include a path",
		},
		{
			name:      "empty endpoint",
			endpoint:  "   ",
			wantError: "endpoint is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			host, secure, err := normalizeS3Endpoint(tt.endpoint, tt.useSSL)
			if tt.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, host)
			assert.Equal(t, tt.wantSSL, secure)
		})
	}
}
