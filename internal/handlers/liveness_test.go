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

func TestLivenessHandler_HandleGet_AlwaysReturns200(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "returns ok on first call"},
		{name: "returns ok on subsequent calls"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			h := &LivenessHandler{}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/healthz", nil)

			// Act
			err := h.HandleGet(gen.PID{}, w, r)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, w.Code)

			var resp dtos.LivenessResponse
			require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
			assert.Equal(t, "ok", resp.Status)
		})
	}
}

func TestLivenessHandler_HandleGet_ContentType(t *testing.T) {
	// Arrange
	h := &LivenessHandler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	// Act
	_ = h.HandleGet(gen.PID{}, w, r)

	// Assert
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}
