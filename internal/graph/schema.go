package graph

// JSONSchema returns the schema for the graph collection in a JSON schema (https://json-schema.org/)
func JSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type": "string",
			},
			"root": map[string]interface{}{
				"type":       "object",
				"properties": nodeSchema(),
				"required":   []string{"id", "provider", "edge"},
			},
			"nodes": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":       "object",
					"properties": nodeSchema(),
					"required":   []string{"id", "provider", "edge"},
				},
				"minItems": 1,
			},
		},
		"required": []string{"id", "root", "nodes"},
	}
}

// nodeSchema returns the schema for the node collection in a JSON
func nodeSchema() map[string]interface{} {
	return map[string]interface{}{
		"id": map[string]interface{}{
			"type": "string",
		},
		"provider": map[string]interface{}{
			"type":       "object",
			"properties": nodeProviderSchema(),
			"required":   []string{"package", "node"},
		},
		"edge": map[string]interface{}{
			"type":       "object",
			"properties": edgeSchema(),
			"required":   []string{"id", "references"},
		},
		"inputs": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type":       "object",
				"properties": nodeInputSchema(),
				"required":   []string{"source", "origin", "mapping"},
			},
			"minItems": 1,
		},
	}
}

// nodeProviderSchema returns the schema for the node provider
func nodeProviderSchema() map[string]interface{} {
	return map[string]interface{}{
		"package": map[string]interface{}{
			"type": "string",
		},
		"node": map[string]interface{}{
			"type": "string",
		},
	}
}

// nodeInputSchema returns the schema for the node input
func nodeInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"source": map[string]interface{}{
			"type": "string",
		},
		"origin": map[string]interface{}{
			"type": []string{"string", "number", "boolean", "null"},
		},
		"mapping": map[string]interface{}{
			"type": "string",
		},
	}
}

// edgeSchema returns the schema for the edge collection in a JSON
func edgeSchema() map[string]interface{} {
	return map[string]interface{}{
		"id": map[string]interface{}{
			"type": "string",
		},
		"references": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"node": map[string]interface{}{
						"type": "string",
					},
					"conditional": map[string]interface{}{
						"type":       "object",
						"properties": edgeConditionalSchema(),
						"required":   []string{"name", "value"},
					},
				},
				"required": []string{"node"},
			},
			"minItems": 1,
		},
	}
}

// edgeConditionalSchema returns the schema for the edge conditional
func edgeConditionalSchema() map[string]interface{} {
	return map[string]interface{}{
		"name": map[string]interface{}{
			"type": "string",
		},
		"value": map[string]interface{}{
			"type": []string{"string", "number", "boolean", "null"},
		},
	}
}
