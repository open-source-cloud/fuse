package ai

import (
	"strings"
	"testing"

	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/stretchr/testify/assert"
)

// msg builds a message of n tokens (~n*4 chars) for a role.
func msg(role llm.Role, n int) llm.Message {
	return llm.Message{Role: role, Content: strings.Repeat("x", n*approxCharsPerToken)}
}

func TestContextStrategyOrDefault(t *testing.T) {
	t.Parallel()
	assert.Equal(t, contextStrategyDropOldest, contextStrategyOrDefault(""))
	assert.Equal(t, contextStrategyDropOldest, contextStrategyOrDefault("nonsense"))
	assert.Equal(t, contextStrategySummarize, contextStrategyOrDefault("summarize"))
}

func TestTrimContext(t *testing.T) {
	t.Parallel()

	// head = system(10) + user(10) = 20 tokens; then 4 middle turns of 10 each.
	build := func() []llm.Message {
		return []llm.Message{
			msg(llm.RoleSystem, 10),
			msg(llm.RoleUser, 10),
			msg(llm.RoleAssistant, 10),
			msg(llm.RoleTool, 10),
			msg(llm.RoleAssistant, 10),
			msg(llm.RoleTool, 10),
		}
	}

	t.Run("budget disabled returns unchanged", func(t *testing.T) {
		t.Parallel()
		in := build()
		kept, dropped := trimContext(in, 0)
		assert.Len(t, kept, len(in))
		assert.Empty(t, dropped)
	})

	t.Run("under budget returns unchanged", func(t *testing.T) {
		t.Parallel()
		in := build() // 60 tokens
		kept, dropped := trimContext(in, 1000)
		assert.Len(t, kept, len(in))
		assert.Empty(t, dropped)
	})

	t.Run("over budget drops oldest middle, preserves head + recent", func(t *testing.T) {
		t.Parallel()
		in := build() // 60 tokens total; head=20
		// budget 40 -> head(20) + at most 2 recent turns (20) fit; drop the 2 oldest middle.
		kept, dropped := trimContext(in, 40)

		// head preserved: first two are the original system + user task.
		assert.Equal(t, llm.RoleSystem, kept[0].Role)
		assert.Equal(t, llm.RoleUser, kept[1].Role)
		// the two most recent turns are retained as the tail.
		assert.Equal(t, in[len(in)-1], kept[len(kept)-1])
		assert.Equal(t, in[len(in)-2], kept[len(kept)-2])
		assert.Len(t, dropped, 2)
		assert.LessOrEqual(t, estimateTokens(kept), 40)
	})

	t.Run("recent turns alone exceeding budget leaves input unchanged", func(t *testing.T) {
		t.Parallel()
		in := []llm.Message{msg(llm.RoleSystem, 10), msg(llm.RoleUser, 100)}
		kept, dropped := trimContext(in, 5)
		assert.Len(t, kept, len(in))
		assert.Empty(t, dropped)
	})
}
