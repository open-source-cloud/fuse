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
)

func TestReadinessHandler_HandleGet_NoPool_AlwaysReady(t *testing.T) {
	// Arrange — no DB pool (memory driver)
	h := &ReadinessHandler{pool: nil}
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
	assert.Empty(t, resp.Checks)
}

func TestReadinessHandler_CheckReadiness_NoPool(t *testing.T) {
	tests := []struct {
		name           string
		pool           any // always nil in this table
		expectedStatus string
		expectedChecks int
	}{
		{
			name:           "memory driver has no DB checks",
			pool:           nil,
			expectedStatus: "ready",
			expectedChecks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			h := &ReadinessHandler{pool: nil}

			// Act
			resp := h.checkReadiness()

			// Assert
			assert.Equal(t, tt.expectedStatus, resp.Status)
			assert.Len(t, resp.Checks, tt.expectedChecks)
		})
	}
}

func TestReadinessHandler_HandleGet_ContentType(t *testing.T) {
	// Arrange
	h := &ReadinessHandler{pool: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	// Act
	_ = h.HandleGet(gen.PID{}, w, r)

	// Assert
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}
