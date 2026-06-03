package llm_test

import (
	"context"
	"testing"

	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubProvider is a minimal Provider for registry tests.
type stubProvider struct{ name string }

func (s stubProvider) Name() string { return s.name }
func (s stubProvider) Chat(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
	return llm.ChatResponse{}, nil
}

func TestRegistry_GetAndDefault(t *testing.T) {
	reg := llm.NewStaticRegistry(map[string]llm.Provider{
		"openai": stubProvider{name: "openai"},
		"ollama": stubProvider{name: "ollama"},
	}, "ollama")

	got, err := reg.Get(context.Background(), "", "openai")
	require.NoError(t, err)
	assert.Equal(t, "openai", got.Name())

	def, err := reg.Default(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "ollama", def.Name())

	assert.ElementsMatch(t, []string{"openai", "ollama"}, reg.List())
}

func TestRegistry_GetUnknownErrors(t *testing.T) {
	reg := llm.NewStaticRegistry(map[string]llm.Provider{}, "")
	_, err := reg.Get(context.Background(), "", "nope")
	assert.ErrorIs(t, err, llm.ErrProviderNotFound)
}

func TestRegistry_NoDefaultErrors(t *testing.T) {
	reg := llm.NewStaticRegistry(map[string]llm.Provider{"openai": stubProvider{name: "openai"}}, "")
	_, err := reg.Default(context.Background(), "")
	assert.ErrorIs(t, err, llm.ErrNoDefaultProvider)
}

// TestRegistry_FactoryDispatchesEnvironment proves the environment is threaded through to the
// factory, so a provider can be built per execution environment (ADR-0031 per-context keys).
func TestRegistry_FactoryDispatchesEnvironment(t *testing.T) {
	reg := llm.NewRegistry(map[string]llm.ProviderFactory{
		"openai": func(_ context.Context, environment string) (llm.Provider, error) {
			return stubProvider{name: "openai-" + environment}, nil
		},
	}, "openai")

	staging, err := reg.Get(context.Background(), "staging", "openai")
	require.NoError(t, err)
	assert.Equal(t, "openai-staging", staging.Name())

	prod, err := reg.Default(context.Background(), "prod")
	require.NoError(t, err)
	assert.Equal(t, "openai-prod", prod.Name())
}
