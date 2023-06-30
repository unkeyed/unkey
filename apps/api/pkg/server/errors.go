package server

type ErrorCode = string

const (
	NOT_FOUND             ErrorCode = "NOT_FOUND"
	BAD_REQUEST           ErrorCode = "BAD_REQUEST"
	UNAUTHORIZED          ErrorCode = "UNAUTHORIZED"
	INTERNAL_SERVER_ERROR ErrorCode = "INTERNAL_SERVER_ERROR"
	RATELIMITED           ErrorCode = "RATELIMITED"
)

type ErrorResponse struct {
	Error string    `json:"error"`
	Code  ErrorCode `json:"code"`
}
