package config_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/stretchr/testify/assert"
)

func TestClusterConfig_PeerNodeNames_Empty(t *testing.T) {
	var c config.ClusterConfig
	assert.Nil(t, c.PeerNodeNames())
}

func TestClusterConfig_PeerNodeNames_TrimsAndSkipsEmpty(t *testing.T) {
	c := config.ClusterConfig{PeerNodesCSV: " a@h1 , ,b@h2 "}
	assert.Equal(t, []string{"a@h1", "b@h2"}, c.PeerNodeNames())
}
