package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyMergeStrategy_Append_ConcatFloat64Slices(t *testing.T) {
	inputs := []BranchInput{
		{EdgeID: "e1", ThreadID: 1, Data: map[string]any{"values": []float64{10}}},
		{EdgeID: "e2", ThreadID: 2, Data: map[string]any{"values": []float64{32}}},
	}

	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeAppend}, inputs)

	val, ok := result["values"].([]float64)
	assert.True(t, ok)
	assert.Equal(t, []float64{10, 32}, val)
}

func TestApplyMergeStrategy_Append_SameKeys(t *testing.T) {
	inputs := []BranchInput{
		{EdgeID: "e1", ThreadID: 1, Data: map[string]any{"count": 10}},
		{EdgeID: "e2", ThreadID: 2, Data: map[string]any{"count": 20}},
	}

	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeAppend}, inputs)

	assert.Equal(t, []any{10, 20}, result["count"])
}

func TestApplyMergeStrategy_Append_DisjointKeys(t *testing.T) {
	inputs := []BranchInput{
		{EdgeID: "e1", ThreadID: 1, Data: map[string]any{"a": 1}},
		{EdgeID: "e2", ThreadID: 2, Data: map[string]any{"b": 2}},
	}

	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeAppend}, inputs)

	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 2, result["b"])
}

func TestApplyMergeStrategy_Append_ThreeBranches(t *testing.T) {
	inputs := []BranchInput{
		{EdgeID: "e1", ThreadID: 1, Data: map[string]any{"x": "a"}},
		{EdgeID: "e2", ThreadID: 2, Data: map[string]any{"x": "b"}},
		{EdgeID: "e3", ThreadID: 3, Data: map[string]any{"x": "c"}},
	}

	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeAppend}, inputs)

	assert.Equal(t, []any{"a", "b", "c"}, result["x"])
}

func TestApplyMergeStrategy_MergeObject(t *testing.T) {
	inputs := []BranchInput{
		{EdgeID: "e1", ThreadID: 1, Data: map[string]any{"key": "first", "a": 1}},
		{EdgeID: "e2", ThreadID: 2, Data: map[string]any{"key": "second", "b": 2}},
	}

	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeObject}, inputs)

	assert.Equal(t, "second", result["key"])
	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 2, result["b"])
}

func TestApplyMergeStrategy_FirstWins(t *testing.T) {
	inputs := []BranchInput{
		{EdgeID: "e1", ThreadID: 1, Data: map[string]any{"winner": "first"}},
		{EdgeID: "e2", ThreadID: 2, Data: map[string]any{"winner": "second"}},
	}

	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeFirstWins}, inputs)

	assert.Equal(t, "first", result["winner"])
}

func TestApplyMergeStrategy_FirstWins_Empty(t *testing.T) {
	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeFirstWins}, []BranchInput{})

	assert.Empty(t, result)
}

func TestApplyMergeStrategy_LastWins(t *testing.T) {
	inputs := []BranchInput{
		{EdgeID: "e1", ThreadID: 1, Data: map[string]any{"winner": "first"}},
		{EdgeID: "e2", ThreadID: 2, Data: map[string]any{"winner": "second"}},
	}

	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeLastWins}, inputs)

	assert.Equal(t, "second", result["winner"])
}

func TestApplyMergeStrategy_LastWins_Empty(t *testing.T) {
	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeLastWins}, []BranchInput{})

	assert.Empty(t, result)
}

func TestApplyMergeStrategy_Keyed(t *testing.T) {
	inputs := []BranchInput{
		{EdgeID: "branch-a", ThreadID: 1, Data: map[string]any{"result": "alpha"}},
		{EdgeID: "branch-b", ThreadID: 2, Data: map[string]any{"result": "beta"}},
	}

	result := ApplyMergeStrategy(MergeConfig{Strategy: MergeKeyed}, inputs)

	branchA, ok := result["branch-a"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "alpha", branchA["result"])

	branchB, ok := result["branch-b"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "beta", branchB["result"])
}

func TestDefaultMergeConfig(t *testing.T) {
	config := DefaultMergeConfig()

	assert.Equal(t, MergeAppend, config.Strategy)
}

func TestApplyMergeStrategy_UnknownFallsBackToAppend(t *testing.T) {
	inputs := []BranchInput{
		{EdgeID: "e1", ThreadID: 1, Data: map[string]any{"v": 1}},
		{EdgeID: "e2", ThreadID: 2, Data: map[string]any{"v": 2}},
	}

	result := ApplyMergeStrategy(MergeConfig{Strategy: "unknown"}, inputs)

	assert.Equal(t, []any{1, 2}, result["v"])
}
