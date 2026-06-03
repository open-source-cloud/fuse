package di

import (
	"context"
	"testing"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvideLLMRegistry_ResolvesKeyPerEnvironment(t *testing.T) {
	ctx := context.Background()
	store := secrets.NewMemorySecretStore()
	require.NoError(t, store.Set(ctx, secrets.Scope{Environment: "staging"}, "openai_key", "sk-staging"))

	cfg := &config.Config{
		Environment: "default",
		LLM: config.LLMConfig{
			DefaultProvider: providerOpenAI,
			OpenAI: config.LLMProviderConfig{
				Enabled: true,
				APIKey:  "{{secret:openai_key}}",
				Model:   "gpt-4o-mini",
			},
		},
	}

	reg := provideLLMRegistry(cfg, store)

	// The secret resolves in the staging environment -> provider builds.
	prov, err := reg.Get(ctx, "staging", providerOpenAI)
	require.NoError(t, err)
	assert.Equal(t, providerOpenAI, prov.Name())

	// An environment without the secret surfaces a resolution error (not a panic).
	_, err = reg.Get(ctx, "prod", providerOpenAI)
	require.Error(t, err)
	assert.ErrorIs(t, err, secrets.ErrSecretNotFound)
}

func TestProvideLLMRegistry_ResolvesCredential(t *testing.T) {
	ctx := context.Background()
	store := secrets.NewMemorySecretStore()
	// A credential's apiKey field value lives at the reserved cred/<id>/apiKey secret name.
	require.NoError(t, store.Set(ctx, secrets.Scope{Environment: "staging"},
		secrets.CredentialSecretName("openai-prod", "apiKey"), "sk-from-credential"))

	cfg := &config.Config{
		Environment: "default",
		LLM: config.LLMConfig{
			DefaultProvider: providerOpenAI,
			OpenAI: config.LLMProviderConfig{
				Enabled:    true,
				Credential: "openai-prod",
				Model:      "gpt-4o-mini",
			},
		},
	}

	reg := provideLLMRegistry(cfg, store)

	// The credential's apiKey resolves in staging -> provider builds.
	prov, err := reg.Get(ctx, "staging", providerOpenAI)
	require.NoError(t, err)
	assert.Equal(t, providerOpenAI, prov.Name())

	// An environment lacking the credential value surfaces a resolution error, not a panic.
	_, err = reg.Get(ctx, "prod", providerOpenAI)
	require.Error(t, err)
	assert.ErrorIs(t, err, secrets.ErrSecretNotFound)
}

func TestProvideLLMRegistry_StaticProviderIsSingleton(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Environment: "default",
		LLM: config.LLMConfig{
			DefaultProvider: providerOllama,
			Ollama: config.LLMProviderConfig{
				Enabled: true,
				BaseURL: "http://localhost:11434/v1",
				Model:   "llama3",
			},
		},
	}

	reg := provideLLMRegistry(cfg, secrets.NewMemorySecretStore())

	// No secret refs -> the provider is built once and reused (fast path), regardless of env.
	p1, err := reg.Get(ctx, "staging", providerOllama)
	require.NoError(t, err)
	p2, err := reg.Get(ctx, "prod", providerOllama)
	require.NoError(t, err)
	assert.Same(t, p1, p2)
}
