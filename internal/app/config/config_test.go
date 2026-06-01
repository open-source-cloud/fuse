package config_test

import (
	"testing"

	"github.com/caarlos0/env/v11"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterConfig_PeerNodeNames_Empty(t *testing.T) {
	var c config.ClusterConfig
	assert.Nil(t, c.PeerNodeNames())
}

func TestClusterConfig_PeerNodeNames_TrimsAndSkipsEmpty(t *testing.T) {
	c := config.ClusterConfig{PeerNodesCSV: " a@h1 , ,b@h2 "}
	assert.Equal(t, []string{"a@h1", "b@h2"}, c.PeerNodeNames())
}

func TestClusterConfig_EtcdEndpointsList_Empty(t *testing.T) {
	var c config.ClusterConfig
	assert.Nil(t, c.EtcdEndpointsList())
}

func TestClusterConfig_EtcdEndpointsList_Trims(t *testing.T) {
	c := config.ClusterConfig{EtcdEndpointsCSV: " http://a:2379 , http://b:2379 "}
	assert.Equal(t, []string{"http://a:2379", "http://b:2379"}, c.EtcdEndpointsList())
}

func TestClusterConfig_DiscoveryModeNormalized_Default(t *testing.T) {
	var c config.ClusterConfig
	assert.Equal(t, config.ClusterDiscoveryModeStatic, c.DiscoveryModeNormalized())
	etcdMode := config.ClusterConfig{DiscoveryMode: "ETCD"}
	assert.Equal(t, config.ClusterDiscoveryModeEtcd, etcdMode.DiscoveryModeNormalized())
}

func TestConfig_Validate_EtcdRequiresEndpoints(t *testing.T) {
	c := &config.Config{
		Cluster: config.ClusterConfig{
			Enabled:       true,
			DiscoveryMode: config.ClusterDiscoveryModeEtcd,
		},
	}
	assert.Error(t, c.Validate())

	c.Cluster.EtcdEndpointsCSV = "http://localhost:2379"
	assert.NoError(t, c.Validate())
}

func TestLLMConfig_EnvPrefixParsing(t *testing.T) {
	t.Setenv("LLM_DEFAULT_PROVIDER", "openai")
	t.Setenv("LLM_OLLAMA_ENABLED", "true")
	t.Setenv("LLM_OLLAMA_BASE_URL", "http://localhost:11434/v1")
	t.Setenv("LLM_OLLAMA_MODEL", "llama3.1")
	t.Setenv("LLM_OPENAI_ENABLED", "true")
	t.Setenv("LLM_OPENAI_API_KEY", "sk-test")
	t.Setenv("LLM_OPENAI_MODEL", "gpt-4o")
	t.Setenv("LLM_OPENAI_TEMPERATURE", "0.2")

	var cfg config.LLMConfig
	require.NoError(t, env.Parse(&cfg))

	assert.Equal(t, "openai", cfg.DefaultProvider)

	assert.True(t, cfg.Ollama.Enabled)
	assert.Equal(t, "http://localhost:11434/v1", cfg.Ollama.BaseURL)
	assert.Equal(t, "llama3.1", cfg.Ollama.Model)

	assert.True(t, cfg.OpenAI.Enabled)
	assert.Equal(t, "sk-test", cfg.OpenAI.APIKey)
	assert.Equal(t, "gpt-4o", cfg.OpenAI.Model)
	assert.InDelta(t, 0.2, cfg.OpenAI.Temperature, 0.0001)

	// Unset providers default to disabled.
	assert.False(t, cfg.Anthropic.Enabled)
}
