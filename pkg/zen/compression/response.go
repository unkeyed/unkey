// Package compression writes HTTP responses using pooled gzip and zstd encoders.
package compression

import (
	"net/http"
	"strconv"

	"github.com/unkeyed/unkey/pkg/fault"
)

// WriteResponse sends body with status 200 using gzip or zstd. Unsupported
// encodings and bodies that do not shrink are sent unchanged. Callers must
// negotiate encoding and check whether the response permits compression.
func WriteResponse(w http.ResponseWriter, body []byte, encoding string) error {
	switch encoding {
	case "gzip":
		return writeGzip(w, body)
	case "zstd":
		return writeZstd(w, body)
	default:
		return writeBody(w, body)
	}
}

func writeEncoded(w http.ResponseWriter, body, encoded []byte, encoding string) error {
	if len(encoded) >= len(body) {
		return writeBody(w, body)
	}
	w.Header().Set("Content-Encoding", encoding)
	w.Header().Set("Content-Length", strconv.Itoa(len(encoded)))
	return writeBody(w, encoded)
}

func writeBody(w http.ResponseWriter, body []byte) error {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		return fault.Wrap(err, fault.Internal("failed to send bytes"), fault.Public("Unable to send response body."))
	}
	return nil
}
