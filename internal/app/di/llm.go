package di

import (
	"context"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/llm/providers/anthropic"
	"github.com/open-source-cloud/fuse/internal/llm/providers/openaicompat"
	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// Provider registry keys. These are the names agents reference via the
// "provider" input, and the values matched against LLM_DEFAULT_PROVIDER.
const (
	providerOpenAI     = "openai"
	providerOpenRouter = "openrouter"
	providerOllama     = "ollama"
	providerGemini     = "gemini"
	providerAnthropic  = "anthropic"
)

// LLMModule provides the LLM provider registry built from configuration.
var LLMModule = fx.Module(
	"llm",
	fx.Provide(provideLLMRegistry),
)

type llmProvider struct {
	name string
	conf config.LLMProviderConfig
}

// providerBuilder constructs a provider from already-resolved connection values.
type providerBuilder func(name, apiKey, baseURL, model string) llm.Provider

// provideLLMRegistry builds an llm.Registry of per-provider factories from configuration. A
// provider's APIKey / BaseURL may be a {{secret:NAME}} reference (ADR-0031), in which case its
// factory resolves the reference from the SecretStore against the running workflow's environment
// on each call; providers with fully-static config are built once (fast path). All providers are
// disabled by default; only enabled ones are registered.
func provideLLMRegistry(cfg *config.Config, secretStore secrets.SecretStore) llm.Registry {
	factories := make(map[string]llm.ProviderFactory)

	openAICompatBuild := func(name, apiKey, baseURL, model string) llm.Provider {
		return openaicompat.New(openaicompat.Config{Name: name, APIKey: apiKey, BaseURL: baseURL, Model: model})
	}
	anthropicBuild := func(name, apiKey, baseURL, model string) llm.Provider {
		return anthropic.New(anthropic.Config{Name: name, APIKey: apiKey, BaseURL: baseURL, Model: model})
	}

	// OpenAI-compatible providers share one implementation, differing only by base URL + key.
	openAICompat := []llmProvider{
		{name: providerOpenAI, conf: cfg.LLM.OpenAI},
		{name: providerOpenRouter, conf: cfg.LLM.OpenRouter},
		{name: providerOllama, conf: cfg.LLM.Ollama},
		{name: providerGemini, conf: cfg.LLM.Gemini},
	}
	for _, p := range openAICompat {
		if !p.conf.Enabled {
			continue
		}
		factories[p.name] = newProviderFactory(p.name, p.conf, openAICompatBuild, secretStore, cfg.Environment)
		log.Info().Str("provider", p.name).Str("model", p.conf.Model).Msg("LLM provider registered")
	}

	// Anthropic uses its own native protocol, so it has a separate implementation.
	if cfg.LLM.Anthropic.Enabled {
		factories[providerAnthropic] = newProviderFactory(providerAnthropic, cfg.LLM.Anthropic, anthropicBuild, secretStore, cfg.Environment)
		log.Info().Str("provider", providerAnthropic).Str("model", cfg.LLM.Anthropic.Model).Msg("LLM provider registered")
	}

	if len(factories) == 0 {
		log.Info().Msg("no LLM providers enabled; ai/chat and ai/agent nodes will be unavailable")
	}

	return llm.NewRegistry(factories, cfg.LLM.DefaultProvider)
}

// newProviderFactory returns a factory for one provider. When the config has no {{secret:NAME}}
// references the provider is built once and the factory returns that singleton (preserving the
// previous static behavior). Otherwise the factory resolves the references per call against the
// given environment (falling back to defaultEnv when empty) and builds a fresh provider.
func newProviderFactory(name string, conf config.LLMProviderConfig, build providerBuilder, store secrets.SecretStore, defaultEnv string) llm.ProviderFactory {
	if !secrets.HasSecretRef(conf.APIKey) && !secrets.HasSecretRef(conf.BaseURL) {
		p := build(name, conf.APIKey, conf.BaseURL, conf.Model)
		return func(_ context.Context, _ string) (llm.Provider, error) { return p, nil }
	}

	return func(ctx context.Context, environment string) (llm.Provider, error) {
		env := environment
		if env == "" {
			env = defaultEnv
		}
		resolve := func(refName string) (string, error) {
			v, err := store.Resolve(ctx, secrets.Scope{Environment: env}, refName)
			if err != nil {
				return "", err
			}
			return v.Reveal(), nil
		}
		apiKey, err := secrets.ReplaceSecretRefs(conf.APIKey, resolve)
		if err != nil {
			return nil, fmt.Errorf("llm[%s]: resolve api key for environment %q: %w", name, env, err)
		}
		baseURL, err := secrets.ReplaceSecretRefs(conf.BaseURL, resolve)
		if err != nil {
			return nil, fmt.Errorf("llm[%s]: resolve base url for environment %q: %w", name, env, err)
		}
		return build(name, apiKey, baseURL, conf.Model), nil
	}
}
