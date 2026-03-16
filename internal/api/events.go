package api

// EventEmitter is the interface for emitting events to the frontend.
type EventEmitter interface {
	Emit(eventName string, data ...interface{})
}

// NoopEmitter is an EventEmitter that does nothing, useful for tests.
type NoopEmitter struct{}

// Emit does nothing.
func (NoopEmitter) Emit(eventName string, data ...interface{}) {}
