package transport

import "errors"

var (
	errNilNode          = errors.New("transport: nil node for async function result")
	errNilExecutionInfo = errors.New("transport: nil ExecutionInfo")
)
