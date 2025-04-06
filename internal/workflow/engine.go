package workflow

import (
	"github.com/rs/zerolog/log"
	"sync"
)

type Engine interface {
	Run() error
	AddProvider(providerSpec NodeProviderSpec) error
	AddSchema(schema Schema) error
	ExecuteWorkflow(workflowId string, input interface{}) error
}

type EngineExecuteSignal struct {
	Signal string
	Data   interface{}
}

type DefaultEngine struct {
	executeChan chan EngineExecuteSignal
	state       State
}

// NewDefaultEngine is a constructor for DefaultEngine.
func NewDefaultEngine(state State) *DefaultEngine {
	return &DefaultEngine{
		state: state,
	}
}

func (e *DefaultEngine) Run() error {
	log.Info().Msg("Engine started")

	var wg sync.WaitGroup
	e.executeChan = make(chan EngineExecuteSignal)

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
				if signal.Signal == "workflow-execute" {
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

	e.executeChan <- EngineExecuteSignal{Signal: "started", Data: nil}

	wg.Wait()
	return nil
}

func (e *DefaultEngine) AddProvider(providerSpec NodeProviderSpec) error {
	provider := NewDefaultNodeProvider(providerSpec)
	return e.state.AddProvider(provider)
}

func (e *DefaultEngine) AddSchema(schema Schema) error {
	return e.state.AddSchema(schema)
}

func (e *DefaultEngine) ExecuteWorkflow(workflowId string, input interface{}) error {
	// Log workflow execution initiation
	log.Info().Msgf("Executing workflow with ID: %s", workflowId)

	e.executeChan <- EngineExecuteSignal{Signal: "workflow-execute", Data: struct {
		WorkflowId string
		Input      interface{}
	}{
		WorkflowId: workflowId,
		Input:      input,
	}}
	return nil
}

func (e *DefaultEngine) handleExecuteWorkflow(workflowId string, input interface{}) {
	log.Info().Msgf("Executing workflow with ID: %s", workflowId)
}
