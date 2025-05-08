package actors

import (
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

const EngineMeta = "engine_meta"

func NewEngineMeta(engine workflow.Engine) gen.MetaBehavior {
	return &engineMeta{
		engine: engine,
	}
}

type engineMeta struct {
	gen.MetaProcess
	engine workflow.Engine
}

func (m *engineMeta) Init(meta gen.MetaProcess) error {
	m.MetaProcess = meta
	return nil
}

func (m *engineMeta) Start() error {
	m.Log().Info("starting '%s' process", HttpServerMeta)

	//defer func() {
	//	err := m.server.Shutdown()
	//	if err != nil {
	//		m.Log().Error("Failed to shutdown server : %s", err)
	//		return
	//	}
	//}()
	//m.server = server.New(m.config, m.messageChan)
	//
	//for v := range m.messageChan {
	//	err := m.Send(m.Parent(), v)
	//	if err != nil {
	//		m.Log().Error("Failed to send message : %s", err)
	//	}
	//}

	return nil
}

func (m *engineMeta) HandleMessage(from gen.PID, message any) error {
	return nil
}

func (m *engineMeta) HandleCall(from gen.PID, ref gen.Ref, request any) (any, error) {
	return nil, nil
}

func (m *engineMeta) Terminate(reason error) {}

func (m *engineMeta) HandleInspect(from gen.PID, item ...string) map[string]string {
	return nil
}
