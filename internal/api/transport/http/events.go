package httptransport

// SSE event names delivered to clients on /api/v1/events.
const (
	EventTaskCreated   = "task.created"
	EventTaskUpdated   = "task.updated"
	EventAgentLog      = "agent.log"
	EventAgentToolCall = "agent.tool_call"
	EventAgentFinished = "agent.finished"
	// EventMessage is the fallback name for a frame without an event field.
	EventMessage = "message"
)

// Envelope keys written by Hub.Publish and read back by the SSE handler.
const (
	FieldEvent   = "event"
	FieldPayload = "payload"
)
