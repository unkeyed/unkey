package httpclient

import "fmt"

// Error represents a non-successful HTTP response.
type Error struct {
	StatusCode int
	Body       []byte
}

func (e *Error) Error() string {
	return fmt.Sprintf("http %d: %s", e.StatusCode, string(e.Body))
}
