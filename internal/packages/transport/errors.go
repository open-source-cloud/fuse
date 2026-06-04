package transport

import "errors"

var (
	errNilNode          = errors.New("transport: nil node for async function result")
	errNilExecutionInfo = errors.New("transport: nil ExecutionInfo")
	// errNilFunction guards against executing an Internal transport whose function pointer is nil.
	// This happens when an internal package is decoded from persistence (PackagedFunction.Function
	// is json:"-", so it is lost) and registered as executable; calling it would panic the worker.
	errNilFunction = errors.New("transport: nil function (internal package likely loaded from persistence without its code-backed function)")
)
