package workflow

import (
	"maps"
	"reflect"
)

// MergeStrategyType defines how to combine outputs from parallel branches at a join node
type MergeStrategyType string

const (
	// MergeAppend appends all values into arrays (current default behavior)
	MergeAppend MergeStrategyType = "append"
	// MergeObject merges all outputs into a single map (later values override earlier)
	MergeObject MergeStrategyType = "merge"
	// MergeFirstWins uses the first branch's output that completed
	MergeFirstWins MergeStrategyType = "first"
	// MergeLastWins uses the last branch's output that completed
	MergeLastWins MergeStrategyType = "last"
	// MergeKeyed groups outputs by branch edge ID into a keyed map
	MergeKeyed MergeStrategyType = "keyed"
)

// MergeConfig defines the merge strategy for a join node
type MergeConfig struct {
	Strategy MergeStrategyType `json:"strategy" yaml:"strategy" validate:"oneof=append merge first last keyed"`
}

// DefaultMergeConfig returns the default merge strategy (append, matches current behavior)
func DefaultMergeConfig() MergeConfig {
	return MergeConfig{Strategy: MergeAppend}
}

// BranchInput represents the output from one branch arriving at a join node
type BranchInput struct {
	EdgeID   string
	ThreadID uint16
	Data     map[string]any
}

// ApplyMergeStrategy combines inputs from multiple parent edges using the specified strategy
func ApplyMergeStrategy(config MergeConfig, inputs []BranchInput) map[string]any {
	switch config.Strategy {
	case MergeObject:
		return mergeObjects(inputs)
	case MergeFirstWins:
		return mergeFirstWins(inputs)
	case MergeLastWins:
		return mergeLastWins(inputs)
	case MergeKeyed:
		return mergeKeyed(inputs)
	default:
		return mergeAppend(inputs)
	}
}

func mergeAppend(inputs []BranchInput) map[string]any {
	result := make(map[string]any)
	for _, input := range inputs {
		for key, value := range input.Data {
			if existing, ok := result[key]; ok {
				if merged, did := mergeAppendCompatibleSlices(existing, value); did {
					result[key] = merged
					continue
				}
				switch v := existing.(type) {
				case []any:
					result[key] = append(v, value)
				default:
					result[key] = []any{v, value}
				}
			} else {
				result[key] = value
			}
		}
	}
	return result
}

// mergeAppendCompatibleSlices concatenates two values when both are non-empty slices of the same element type.
func mergeAppendCompatibleSlices(existing, value any) (any, bool) {
	ev := reflect.ValueOf(existing)
	vv := reflect.ValueOf(value)
	if ev.Kind() != reflect.Slice || vv.Kind() != reflect.Slice {
		return nil, false
	}
	if ev.Type() != vv.Type() {
		return nil, false
	}
	out := reflect.MakeSlice(ev.Type(), ev.Len()+vv.Len(), ev.Len()+vv.Len())
	reflect.Copy(out.Slice(0, ev.Len()), ev)
	reflect.Copy(out.Slice(ev.Len(), out.Len()), vv)
	return out.Interface(), true
}

func mergeObjects(inputs []BranchInput) map[string]any {
	result := make(map[string]any)
	for _, input := range inputs {
		maps.Copy(result, input.Data)
	}
	return result
}

func mergeFirstWins(inputs []BranchInput) map[string]any {
	if len(inputs) == 0 {
		return make(map[string]any)
	}
	return inputs[0].Data
}

func mergeLastWins(inputs []BranchInput) map[string]any {
	if len(inputs) == 0 {
		return make(map[string]any)
	}
	return inputs[len(inputs)-1].Data
}

func mergeKeyed(inputs []BranchInput) map[string]any {
	result := make(map[string]any)
	for _, input := range inputs {
		result[input.EdgeID] = input.Data
	}
	return result
}
