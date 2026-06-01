package di

import (
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/llm/providers/openaicompat"
	"github.com/open-source-cloud/fuse/pkg/llm"
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

// provideLLMRegistry builds an llm.Registry from the configured providers.
// All providers are disabled by default; only enabled ones are registered.
func provideLLMRegistry(cfg *config.Config) llm.Registry {
	providers := make(map[string]llm.Provider)

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
		providers[p.name] = openaicompat.New(openaicompat.Config{
			Name:    p.name,
			APIKey:  p.conf.APIKey,
			BaseURL: p.conf.BaseURL,
			Model:   p.conf.Model,
		})
		log.Info().Str("provider", p.name).Str("model", p.conf.Model).Msg("LLM provider registered")
	}

	// Anthropic (native protocol) is added in Phase C.
	if cfg.LLM.Anthropic.Enabled {
		log.Warn().Msg("LLM_ANTHROPIC_ENABLED is set but the Anthropic provider is not yet implemented (Phase C)")
	}

	if len(providers) == 0 {
		log.Info().Msg("no LLM providers enabled; ai/chat and ai/agent nodes will be unavailable")
	}

	return llm.NewRegistry(providers, cfg.LLM.DefaultProvider)
}
