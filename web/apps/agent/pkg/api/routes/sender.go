package routes

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/unkeyed/unkey/svc/agent/pkg/api/ctxutil"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/openapi"
)

type Sender interface {

	// Send marshals the body and sends it as a response with the given status code.
	// If marshalling fails, it will return a 500 response with the error message.
	Send(ctx context.Context, w http.ResponseWriter, status int, body any)
}

type JsonSender struct {
	logger logging.Logger
}

func NewJsonSender(logger logging.Logger) Sender {
	return &JsonSender{logger: logger}
}

// Send returns a JSON response with the given status code and body.
// If marshalling fails, it will return a 500 response with the error message.
func (r *JsonSender) Send(ctx context.Context, w http.ResponseWriter, status int, body any) {
	if body == nil {
		return
	}

	b, err := json.Marshal(body)
	if err != nil {
		r.logger.Error().Err(err).Interface("body", body).Msg("failed to marshal response body")
		w.WriteHeader(http.StatusInternalServerError)

		error := openapi.BaseError{
			Title:     "Internal Server Error",
			Detail:    "failed to marshal response body",
			Instance:  "https://errors.unkey.com/todo",
			Status:    http.StatusInternalServerError,
			RequestId: ctxutil.GetRequestId(ctx),
			Type:      "TODO docs link",
		}

		b, err = json.Marshal(error)
		if err != nil {
			_, err = w.Write([]byte("failed to marshal response body"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		_, err = w.Write(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
