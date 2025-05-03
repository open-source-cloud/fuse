package graph

// JSONSchema returns the schema for the graph collection in a JSON schema (https://json-schema.org/)
func JSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"$id":         "https://fuse.uranus.com.br/api/v1/schemas",
		"title":       "FUSE Workflow Engine Graph Schema",
		"description": "FUSE Workflow Engine Graph Schema",
		"type":        "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the graph. Must be unique for your workflow graph schema.",
			},
			"graph": WorkflowSchema(),
			"version": map[string]interface{}{
				"type":        "string",
				"description": "The version of your schema.",
			},
		},
	}
}

// WorkflowSchema returns the schema for the graph collection in a JSON
func WorkflowSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": "Workflow schema",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the workflow. Must be unique for your workflow graph schema, (e.g sum-rand-branch or a UUID v7)",
			},
			"root": RootNodeSchema(),
			"nodes": map[string]interface{}{
				"type":        "array",
				"description": "The nodes of the workflow",
				"items":       NodeSchema(),
				"minItems":    1,
			},
		},
		"required": []string{"id", "root", "nodes"},
	}
}

// RootNodeSchema returns the schema for the root node collection in a JSON
func RootNodeSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": "Root node schema",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the root node. Must be unique for your workflow graph schema, (e.g debug-nil or a UUID v7)",
			},
			"provider": NodeProviderSchema(),
		},
		"required": []string{"id", "provider"},
	}
}

// NodeSchema returns the schema for the node collection in a JSON
func NodeSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type": "string",
			},
			"provider": NodeProviderSchema(),
			"edge":     EdgeSchema(),
			"inputs": map[string]interface{}{
				"type":     "array",
				"items":    NodeInputSchema(),
				"minItems": 1,
			},
		},
		"required": []string{"id", "provider", "edge"},
	}
}

// NodeProviderSchema returns the schema for the node provider
func NodeProviderSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": "Node provider schema",
		"properties": map[string]interface{}{
			"package": map[string]interface{}{
				"type":        "string",
				"description": "The package name of the node provider. (e.g fuse/internal/providers/logic)",
			},
			"node": map[string]interface{}{
				"type":        "string",
				"description": "The node name of the node provider. (e.g fuse/internal/providers/logic/timer)",
			},
		},
		"required": []string{"package", "node"},
	}
}

// NodeInputSchema returns the schema for the node input
func NodeInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"source": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"schema", "edge"},
				"description": "The source of the input.",
				"default":     "schema",
			},
			"origin": map[string]interface{}{
				"type":        []string{"string", "number", "boolean", "null"},
				"description": "The origin of the input.",
			},
			"mapping": map[string]interface{}{
				"type":        "string",
				"description": "The mapping of the input. (e.g data.name)",
			},
		},
		"required": []string{"source", "origin", "mapping"},
	}
}

// EdgeSchema returns the schema for the edge collection in a JSON
func EdgeSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the edge. Must be unique for your workflow graph schema, (e.g UUID v7)",
			},
			"references": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"node": map[string]interface{}{
							"type":        "string",
							"description": "The ID of the node that the edge references. (e.g sum-rand-branch)",
						},
						"conditional": EdgeConditionalSchema(),
					},
					"required": []string{"node"},
				},
				"minItems": 1,
			},
		},
		"required": []string{"id"},
	}
}

// EdgeConditionalSchema returns the schema for the edge conditional
func EdgeConditionalSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the conditional. (e.g sum-rand-branch.sum)",
			},
			"value": map[string]interface{}{
				"type":        []string{"string", "number", "boolean", "null"},
				"description": "The value of the conditional. (e.g 10)",
			},
		},
		"required": []string{"name", "value"},
	}
}
