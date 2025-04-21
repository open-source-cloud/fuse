// Package typeschema for types schema
package typeschema

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ParseValue converts val (interface{}) to the Go type described by typeStr (e.g., "int", "[]string")
func ParseValue(typeStr string, val any) (any, error) {
	typeStr = strings.TrimSpace(typeStr)
	if strings.HasPrefix(typeStr, "[]") {
		elemType := typeStr[2:]
		valSlice, ok := val.([]any)
		if !ok {
			valSlice = []any{val}
		}
		result := make([]any, len(valSlice))
		for i, elem := range valSlice {
			casted, err := ParseValue(elemType, elem)
			if err != nil {
				return nil, fmt.Errorf("invalid element at index %d: %v", i, err)
			}
			result[i] = casted
		}
		return result, nil
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
