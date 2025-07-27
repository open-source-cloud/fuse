// Package typeschema for types schema
package typeschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ParseFunc is a function that converts a value to a specific type
type ParseFunc func(typeStr string, val any) (any, error)

// ParseValue converts val (interface{}) to the Go type described by typeStr (e.g., "int", "[]string")
// For array types, it returns a slice of the actual element type, not []any.
//
//nolint:gocyclo
func ParseValue(typeStr string, val any) (any, error) {
	typeStr = strings.TrimSpace(typeStr)

	if strings.HasPrefix(typeStr, "[]") {
		return handleSlice(typeStr, val)
	}

	mapOfTypes := map[string]ParseFunc{
		"string":         handleString,
		"int":            handleInt,
		"float64":        handleFloat,
		"bool":           handleBool,
		"[]byte":         handleByte,
		"map[string]any": handleMapStringAny,
	}

	parseFunc, ok := mapOfTypes[typeStr]
	if !ok {
		return nil, fmt.Errorf("unsupported type: %s", typeStr)
	}

	return parseFunc(typeStr, val)
}

// mapSlice maps a slice type to the actual element type
func handleSlice(typeStr string, val any) (any, error) {
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

	// Recursively convert items, then use reflection to create the actual slice type
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

	// if elemTypeVal is nil, determine the element type based on the element type
	if elemTypeVal == nil {
		switch elemType {
		case "string":
			elemTypeVal = reflect.TypeOf("")
		case "int":
			elemTypeVal = reflect.TypeOf(0)
		case "float64":
			elemTypeVal = reflect.TypeOf(float64(0))
		case "bool":
			elemTypeVal = reflect.TypeOf(false)
		default:
			return nil, fmt.Errorf("cannot determine array element type for %q", elemType)
		}
	}

	// create the slice with the element type
	resultSlice := reflect.MakeSlice(reflect.SliceOf(elemTypeVal), len(samples), len(samples))
	for i, v := range samples {
		resultSlice.Index(i).Set(v)
	}

	return resultSlice.Interface(), nil
}

// mapString maps a string to the actual type
func handleString(_ string, val any) (any, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// mapInt maps an int to the actual type
func handleInt(_ string, val any) (any, error) {
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
}

// mapFloat maps a float to the actual type
func handleFloat(_ string, val any) (any, error) {
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
}

// mapBool maps a bool to the actual type
func handleBool(_ string, val any) (any, error) {
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
}

// mapByte maps a byte to the actual type
func handleByte(typeStr string, val any) (any, error) {
	byteVal, ok := val.([]byte)
	if !ok {
		return nil, fmt.Errorf("cannot convert to %s, expected []byte from %T", typeStr, val)
	}
	switch typeStr {
	case "string":
		return string(byteVal), nil
	case "[]byte":
		return byteVal, nil
	case "map[string]any":
		var mapVal map[string]any
		err := json.Unmarshal(byteVal, &mapVal)
		if err != nil {
			return nil, fmt.Errorf("cannot convert to %s, %w", typeStr, err)
		}
		return mapVal, nil
	default:
		return nil, fmt.Errorf("cannot convert to %s", typeStr)
	}
}

// mapMapStringAny maps a interface{} to a map[string]any
func handleMapStringAny(typeStr string, val any) (any, error) {
	switch v := val.(type) {
	case map[string]any:
		return v, nil
	case string:
		var mapVal map[string]any
		err := json.Unmarshal([]byte(v), &mapVal)
		if err != nil {
			return nil, fmt.Errorf("cannot convert to %s, %w", typeStr, err)
		}
		return mapVal, nil
	case []byte:
		var mapVal map[string]any
		err := json.Unmarshal(v, &mapVal)
		if err != nil {
			return nil, fmt.Errorf("cannot convert to %s, %w", typeStr, err)
		}
		return mapVal, nil
	default:
		return nil, fmt.Errorf("cannot convert to %s", typeStr)
	}
}
