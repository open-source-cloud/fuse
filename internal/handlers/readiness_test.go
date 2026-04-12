package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ergo.services/ergo/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/readiness"
)

func readyFlag() *readiness.Flag {
	f := readiness.NewFlag()
	f.SetReady()
	return f
}

func TestReadinessHandler_HandleGet_NoPool_AlwaysReady(t *testing.T) {
	// Arrange — no DB pool (memory driver), actors ready
	h := &ReadinessHandler{pool: nil, readinessFlag: readyFlag()}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	// Act
	err := h.HandleGet(gen.PID{}, w, r)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp dtos.ReadinessResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "ready", resp.Status)
	assert.Equal(t, "ok", resp.Checks["actors"])
}

func TestReadinessHandler_CheckReadiness_NoPool(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus string
		expectedChecks int
	}{
		{
			name:           "memory driver has no DB checks",
			expectedStatus: "ready",
			expectedChecks: 1, // actors check only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			h := &ReadinessHandler{pool: nil, readinessFlag: readyFlag()}

			// Act
			resp := h.checkReadiness()

			// Assert
			assert.Equal(t, tt.expectedStatus, resp.Status)
			assert.Len(t, resp.Checks, tt.expectedChecks)
			assert.Equal(t, "ok", resp.Checks["actors"])
		})
	}
}

func TestReadinessHandler_HandleGet_ContentType(t *testing.T) {
	// Arrange
	h := &ReadinessHandler{pool: nil, readinessFlag: readyFlag()}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	// Act
	_ = h.HandleGet(gen.PID{}, w, r)

	// Assert
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestReadinessHandler_HandleGet_NotReady_Returns503(t *testing.T) {
	// Arrange — actors not ready yet
	h := &ReadinessHandler{pool: nil, readinessFlag: readiness.NewFlag()}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	// Act
	err := h.HandleGet(gen.PID{}, w, r)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp dtos.ReadinessResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "not_ready", resp.Status)
	assert.Equal(t, "initializing", resp.Checks["actors"])
}
