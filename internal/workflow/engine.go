package workflow

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
	"sync"
)

type Engine interface {
	RegisterNodeProvider(provider workflow.NodeProvider) error
	Run() error
}

type ExecuteSignal struct {
	Signal string
	Data   interface{}
}

type DefaultEngine struct {
	executeChan chan ExecuteSignal
	schemas     map[string]Schema
	providers   map[string]workflow.NodeProvider
	nodes       map[string]workflow.Node
}

func NewDefaultEngine(executeChan chan ExecuteSignal) *DefaultEngine {
	return &DefaultEngine{
		executeChan: executeChan,
		schemas:     make(map[string]Schema),
		providers:   make(map[string]workflow.NodeProvider),
		nodes:       make(map[string]workflow.Node),
	}
}

func (e *DefaultEngine) RegisterNodeProvider(provider workflow.NodeProvider) error {
	if _, exists := e.providers[provider.ID()]; exists {
		err := fmt.Errorf("provider %s already registered", provider.ID())
		log.Error().Err(err)
		return err
	}
	log.Info().Msgf("Registered provider %s", provider.ID())

	count := 0
	for _, node := range provider.Nodes() {
		if _, exists := e.nodes[node.ID()]; exists {
			err := fmt.Errorf("node %s already registered", node.ID())
			log.Error().Err(err)
			return err
		}
		log.Info().Msgf("  > Registered node %s", node.ID())
		count++
	}

	return nil
}

func (e *DefaultEngine) Run() error {
	log.Info().Msg("Engine started")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case signal := <-e.executeChan:
				if signal.Signal == "quit" {
					log.Info().Msg("Received quit signal, stopping engine")
					return
				}
				// Handle other signals here if needed
				log.Info().Msgf("Received signal: %v", signal.Signal)
				if signal.Signal == "workflow-start" {
					if workflowSchema, ok := signal.Data.(Schema); ok {
						go e.handleStartWorkflow(workflowSchema)
					} else {
						log.Error().Msgf("Invalid workflow-start signal data: %v", signal.Data)
					}
				} else if signal.Signal == "workflow-execute" {
					if signalData, ok := signal.Data.(struct {
						WorkflowId string
						Input      interface{}
					}); ok {
						go e.handleExecuteWorkflow(signalData.WorkflowId, signalData.Input)
					} else {
						log.Error().Msg("Invalid workflow-execute signal data")
					}
				}
			}
		}
	}()

	e.executeChan <- ExecuteSignal{Signal: "started", Data: nil}

	wg.Wait()
	return nil
}

func (e *DefaultEngine) handleStartWorkflow(workflowSchema Schema) {
	log.Info().Msgf("Starting workflow with ID: %s", workflowSchema.ID())

	if _, exists := e.schemas[workflowSchema.ID()]; !exists {
		e.schemas[workflowSchema.ID()] = workflowSchema
	}
	schemaGraph := workflowSchema.Graph()

	// trigger first node
	rootNode := schemaGraph.Root()
	nodeOutput, err := rootNode.NodeRef().Execute(nil)
	if err != nil {
		log.Error().Err(err).Msgf("Workflow#%s > failed to execute root node", workflowSchema.ID())
	}

	e.executeChan <- ExecuteSignal{
		Signal: "workflow-execute",
		Data: struct {
			WorkflowId string
			Input      interface{}
		}{
			WorkflowId: workflowSchema.ID(),
			Input:      nodeOutput,
		},
	}
}

func (e *DefaultEngine) handleExecuteWorkflow(workflowId string, input interface{}) {
	log.Info().Msgf("Executing workflow with ID: %s", workflowId)
}
