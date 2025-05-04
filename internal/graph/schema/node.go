package schema

type (
	// Node represents a structure containing an ID, package details, and optional configuration.
	Node struct {
		ID      string      `json:"id" yaml:"id"`
		Package NodePackage `json:"package" yaml:"package"`
		Config  *NodeConfig `json:"config,omitempty" yaml:"config,omitempty"`
	}
	// NodePackage represents a package definition within a node, including its registry and function details.
	NodePackage struct {
		Registry string `json:"registry" yaml:"registry"`
		Function string `json:"function" yaml:"function"`
	}
	// NodeConfig represents the configuration schema for a node containing a list of input schemas.
	NodeConfig struct {
		Inputs []NodeInputMapping `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	}
	// NodeInputMapping represents a mapping for node input, including source, origin of data, and target mapping name.
	NodeInputMapping struct {
		Source  string `json:"source" yaml:"source"`
		Origin  any    `json:"origin" yaml:"origin"`
		Mapping string `json:"mapping" yaml:"mapping"`
	}
)
