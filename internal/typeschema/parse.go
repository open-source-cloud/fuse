// Package typeschema for types schema
package typeschema

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ParseValue converts val (interface{}) to the Go type described by typeStr (e.g., "int", "[]string")
// For array types, it returns a slice of the actual element type, not []any.
func ParseValue(typeStr string, val any) (any, error) {
	typeStr = strings.TrimSpace(typeStr)
	if strings.HasPrefix(typeStr, "[]") {
		elemType := typeStr[2:]
		var valSlice []any
		switch v := val.(type) {
		case []any:
			valSlice = v
		default:
			// Try to convert common slice types to []any, or wrap standalone value
			rv := reflect.ValueOf(val)
			if rv.Kind() == reflect.Slice {
				valSlice = make([]any, rv.Len())
				for i := 0; i < rv.Len(); i++ {
					valSlice[i] = rv.Index(i).Interface()
				}
			} else {
				valSlice = []any{val}
			}
		}
		// Recursively convert items, then use reflect to create the actual slice type
		samples := make([]reflect.Value, len(valSlice))
		var elemTypeVal reflect.Type
		for i, item := range valSlice {
			casted, err := ParseValue(elemType, item)
			if err != nil {
				return nil, fmt.Errorf("invalid element at index %d: %v", i, err)
			}
			samples[i] = reflect.ValueOf(casted)
			if i == 0 {
				elemTypeVal = samples[0].Type()
			}
		}
		if elemTypeVal == nil {
			switch elemType {
			case "string":
				elemTypeVal = reflect.TypeOf("")
			case "int":
				elemTypeVal = reflect.TypeOf(int(0))
			case "float64":
				elemTypeVal = reflect.TypeOf(float64(0))
			case "bool":
				elemTypeVal = reflect.TypeOf(false)
			default:
				return nil, fmt.Errorf("cannot determine array element type for %q", elemType)
			}
		}
		resultSlice := reflect.MakeSlice(reflect.SliceOf(elemTypeVal), len(samples), len(samples))
		for i, v := range samples {
			resultSlice.Index(i).Set(v)
		}
		return resultSlice.Interface(), nil
	}
	// Handle scalar types
	switch typeStr {
	case "string":
		switch v := val.(type) {
		case string:
			return v, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}
	case "int":
		switch v := val.(type) {
		case int:
			return v, nil
		case int32:
			return int(v), nil
		case int64:
			return int(v), nil
		case float64:
			return int(v), nil
		case float32:
			return int(v), nil
		case string:
			i, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			return i, nil
		default:
			return nil, errors.New("cannot convert to int")
		}
	case "float64":
		switch v := val.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case int32:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			return f, nil
		default:
			return nil, errors.New("cannot convert to float64")
		}
	case "bool":
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			return b, nil
		default:
			return nil, errors.New("cannot convert to bool")
		}
	default:
		return nil, fmt.Errorf("unsupported type: %s", typeStr)
	}
}
