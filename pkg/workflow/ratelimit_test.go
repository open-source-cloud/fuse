package workflow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration_MarshalJSON(t *testing.T) {
	d := Duration(5 * time.Second)
	data, err := json.Marshal(d)
	require.NoError(t, err)
	assert.Equal(t, `"5s"`, string(data))
}

func TestDuration_UnmarshalJSON_String(t *testing.T) {
	var d Duration
	err := json.Unmarshal([]byte(`"30s"`), &d)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, d.TimeDuration())
}

func TestDuration_UnmarshalJSON_ComplexString(t *testing.T) {
	var d Duration
	err := json.Unmarshal([]byte(`"1h30m"`), &d)
	require.NoError(t, err)
	assert.Equal(t, 90*time.Minute, d.TimeDuration())
}

func TestDuration_UnmarshalJSON_Nanoseconds(t *testing.T) {
	var d Duration
	err := json.Unmarshal([]byte(`1000000000`), &d) // 1 second in nanoseconds
	require.NoError(t, err)
	assert.Equal(t, 1*time.Second, d.TimeDuration())
}

func TestDuration_UnmarshalJSON_InvalidString(t *testing.T) {
	var d Duration
	err := json.Unmarshal([]byte(`"notaduration"`), &d)
	assert.Error(t, err)
}

func TestDuration_RoundTrip(t *testing.T) {
	original := Duration(15 * time.Minute)
	data, err := json.Marshal(original)
	require.NoError(t, err)

	var parsed Duration
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, original, parsed)
}

func TestRateLimitConfig_JSON_RoundTrip(t *testing.T) {
	cfg := RateLimitConfig{
		Limit:    100,
		Period:   Duration(1 * time.Hour),
		Key:      "input.apiKey",
		Strategy: RateLimitReject,
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var parsed RateLimitConfig
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, cfg.Limit, parsed.Limit)
	assert.Equal(t, cfg.Period, parsed.Period)
	assert.Equal(t, cfg.Key, parsed.Key)
	assert.Equal(t, cfg.Strategy, parsed.Strategy)
}

func TestRateLimitConfig_JSON_DefaultStrategy(t *testing.T) {
	jsonData := `{"limit":10,"period":"1m"}`
	var cfg RateLimitConfig
	err := json.Unmarshal([]byte(jsonData), &cfg)
	require.NoError(t, err)

	assert.Equal(t, 10, cfg.Limit)
	assert.Equal(t, Duration(1*time.Minute), cfg.Period)
	assert.Equal(t, RateLimitStrategy(""), cfg.Strategy) // default empty
}
