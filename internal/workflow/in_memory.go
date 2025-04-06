package workflow

import (
	"fmt"
	"github.com/rs/zerolog/log"
)

type InMemoryState struct {
	providers map[string]NodeProvider
	schemas   map[string]Schema
	nodes     map[string]*NodeSpec
}

// NewInMemoryState creates and returns a new instance of InMemoryState.
func NewInMemoryState() *InMemoryState {
	return &InMemoryState{
		providers: make(map[string]NodeProvider),
		schemas:   make(map[string]Schema),
		nodes:     make(map[string]*NodeSpec),
	}
}

func (m *InMemoryState) AddProvider(provider NodeProvider) error {
	if _, exists := m.providers[provider.ID()]; exists {
		return fmt.Errorf("provider %s already registered", provider.ID())
	}

	m.providers[provider.ID()] = provider
	count := 0
	for _, nodeSpec := range provider.Nodes() {
		err := m.AddNodeSpec(nodeSpec)
		if err != nil {
			return err
		}
		count++
	}

	log.Info().Msgf("Added provider %s with %d nodes", provider.ID(), count)
	return nil
}

func (m *InMemoryState) AddSchema(schema Schema) error {
	if _, exists := m.providers[schema.ID]; exists {
		return fmt.Errorf("provider %s already registered", schema.ID)
	}

	log.Info().Msgf("Added schema %s", schema.ID)
	m.schemas[schema.ID] = schema
	return nil
}

func (m *InMemoryState) AddNodeSpec(nodeSpec NodeSpec) error {
	if _, exists := m.nodes[nodeSpec.ID]; exists {
		return fmt.Errorf("nodeSpec %s already registered", nodeSpec.ID)
	}

	log.Info().Msgf("Added nodeSpec %s", nodeSpec.ID)
	m.nodes[nodeSpec.ID] = &nodeSpec
	return nil
}
