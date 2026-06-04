package actors

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseSpec = `{"swagger":"2.0","host":"localhost:9090","basePath":"/","schemes":["http"]}`

func decodeSpec(t *testing.T, doc string) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(doc), &m))
	return m
}

func TestPatchSwaggerServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		host           string
		tls            bool
		headers        map[string]string
		configBasePath string
		wantHost       string
		wantScheme     string
		wantBasePath   string
	}{
		{
			name:         "plain request uses request host and http",
			host:         "fuse.example.com",
			wantHost:     "fuse.example.com",
			wantScheme:   "http",
			wantBasePath: "/",
		},
		{
			name:         "tls request infers https",
			host:         "fuse.example.com",
			tls:          true,
			wantHost:     "fuse.example.com",
			wantScheme:   "https",
			wantBasePath: "/",
		},
		{
			name:         "forwarded headers win (production behind proxy)",
			host:         "10.0.0.5:9090",
			headers:      map[string]string{"X-Forwarded-Host": "api.prod.example.com", "X-Forwarded-Proto": "https", "X-Forwarded-Prefix": "/fuse"},
			wantHost:     "api.prod.example.com",
			wantScheme:   "https",
			wantBasePath: "/fuse",
		},
		{
			name:         "comma-separated forwarded values take first entry",
			host:         "internal",
			headers:      map[string]string{"X-Forwarded-Host": "edge.example.com, internal.svc", "X-Forwarded-Proto": "https, http"},
			wantHost:     "edge.example.com",
			wantScheme:   "https",
			wantBasePath: "/",
		},
		{
			name:           "configured base path used when no forwarded prefix",
			host:           "fuse.example.com",
			configBasePath: "/fuse",
			wantHost:       "fuse.example.com",
			wantScheme:     "http",
			wantBasePath:   "/fuse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, "/docs/doc.json", nil)
			r.Host = tt.host
			if tt.tls {
				r.TLS = &tls.ConnectionState{}
			}
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}

			got := decodeSpec(t, patchSwaggerServer(baseSpec, r, tt.configBasePath))
			assert.Equal(t, tt.wantHost, got["host"])
			assert.Equal(t, tt.wantBasePath, got["basePath"])
			assert.Equal(t, []any{tt.wantScheme}, got["schemes"])
		})
	}
}

func TestPatchSwaggerServer_InvalidDocReturnedUnchanged(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/docs/doc.json", nil)
	const notJSON = "this is not json"
	assert.Equal(t, notJSON, patchSwaggerServer(notJSON, r, ""))
}
