package workflow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlexibleDuration_UnmarshalJSON_String(t *testing.T) {
	t.Parallel()
	var d FlexibleDuration
	require.NoError(t, json.Unmarshal([]byte(`"750ms"`), &d))
	assert.Equal(t, 750*time.Millisecond, d.Duration())
}

func TestFlexibleDuration_UnmarshalJSON_Nanoseconds(t *testing.T) {
	t.Parallel()
	var d FlexibleDuration
	require.NoError(t, json.Unmarshal([]byte(`1000000`), &d)) // 1ms
	assert.Equal(t, time.Millisecond, d.Duration())
}

func TestFlexibleDuration_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	type row struct {
		D FlexibleDuration `json:"d"`
	}
	b, err := json.Marshal(row{D: FlexibleDuration(2 * time.Second)})
	require.NoError(t, err)
	var out row
	require.NoError(t, json.Unmarshal(b, &out))
	assert.Equal(t, 2*time.Second, out.D.Duration())
}
