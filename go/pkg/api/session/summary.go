package session

import (
	"net/http"
)

type Summary struct {
	Host          string
	Method        string
	Path          string
	RequestHeader http.Header
	RequestBody   []byte

	ResponseStatus int
	ResponseHeader http.Header
	ResponseBody   []byte
}
