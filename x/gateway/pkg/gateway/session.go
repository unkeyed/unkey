package gateway

type Session[SessionContext any] struct {
	ctx       SessionContext
	RequestID string
	GetHeader func(name string) (value string, exists bool)
}
