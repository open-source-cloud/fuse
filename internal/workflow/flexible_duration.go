package workflow

import (
	"encoding/json"
	"fmt"
	"time"
)

// FlexibleDuration is a time.Duration that JSON-decodes from a Go duration string
// (e.g. "300ms", "1s") or from a number of nanoseconds.
type FlexibleDuration time.Duration

// Duration returns the value as time.Duration.
func (d FlexibleDuration) Duration() time.Duration {
	return time.Duration(d)
}

// MarshalJSON encodes as a duration string (e.g. "1s").
func (d FlexibleDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON decodes from a JSON string (time.ParseDuration) or integer nanoseconds.
func (d *FlexibleDuration) UnmarshalJSON(data []byte) error {
	if d == nil {
		return fmt.Errorf("FlexibleDuration: UnmarshalJSON on nil pointer")
	}
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		parsed, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		*d = FlexibleDuration(parsed)
		return nil
	}
	var ns int64
	if err := json.Unmarshal(data, &ns); err != nil {
		return err
	}
	*d = FlexibleDuration(time.Duration(ns))
	return nil
}
