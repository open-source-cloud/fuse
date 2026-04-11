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
			Enabled:         true,
			DiscoveryMode:   config.ClusterDiscoveryModeEtcd,
		},
	}
	assert.Error(t, c.Validate())

	c.Cluster.EtcdEndpointsCSV = "http://localhost:2379"
	assert.NoError(t, c.Validate())
}
