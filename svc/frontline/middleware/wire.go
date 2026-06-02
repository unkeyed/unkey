package middleware

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/publicerr"
)

// protocol identifies the wire format a request expects for errors.
// Detected from request Content-Type and Connect-Protocol-Version.
type protocol int

const (
	// protocolHTTP is plain HTTP. Errors render as HTML or RFC 9457
	// problem+json depending on Accept.
	protocolHTTP protocol = iota

	// protocolGRPC is gRPC over HTTP/2. Errors are trailers-only:
	// HTTP 200, Content-Type: application/grpc, with grpc-status /
	// grpc-message in trailers. Covers application/grpc and its
	// codec subtypes (+proto, +json, +thrift, …).
	protocolGRPC

	// protocolConnectStream is Connect's streaming protocol (unary
	// requests use the unary branch). Errors are HTTP 200 with a
	// single end-stream envelope frame carrying the error.
	protocolConnectStream

	// protocolConnectUnary is Connect's unary HTTP/1.1-friendly
	// protocol. Identified by the Connect-Protocol-Version header.
	// Errors carry an HTTP status mapped from the Connect code and
	// a JSON body {code, message}.
	protocolConnectUnary
)

// detectProtocol inspects the request to pick the wire format. Order
// matters: application/grpc-web shares a prefix with application/grpc
// but isn't supported — it falls back to protocolHTTP so the caller
// at least gets an HTTP envelope rather than a malformed gRPC frame.
func detectProtocol(r *http.Request) protocol {
	ct := r.Header.Get("Content-Type")

	// gRPC-Web (browser) uses a binary in-body trailer frame; we don't
	// emit that yet. Fall through to HTTP so the caller gets JSON.
	if strings.HasPrefix(ct, "application/grpc-web") {
		return protocolHTTP
	}
	if strings.HasPrefix(ct, "application/grpc") {
		return protocolGRPC
	}
	if strings.HasPrefix(ct, "application/connect+") {
		return protocolConnectStream
	}
	if r.Header.Get("Connect-Protocol-Version") != "" {
		return protocolConnectUnary
	}
	return protocolHTTP
}

// writeGRPCError emits a trailers-only gRPC error response. HTTP 200
// always; the gRPC code and message ride the grpc-status / grpc-message
// trailers. Content-Type echoes the request subtype (e.g. +proto, +json)
// so a strict client doesn't reject the response.
//
// Trailers are pre-declared in the Trailer response header and then
// written via the http.TrailerPrefix sentinel — the combination works
// for both HTTP/2 (native trailers) and HTTP/1.1 chunked transfer.
func writeGRPCError(s *zen.Session, problem publicerr.Problem, detail string) error {
	w := s.ResponseWriter()
	h := w.Header()

	ct := s.Request().Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/grpc") ||
		strings.HasPrefix(ct, "application/grpc-web") {
		ct = "application/grpc"
	}
	h.Set("Content-Type", ct)

	// Announce trailers so HTTP/1.1 clients accept them; HTTP/2 picks
	// them up from the TrailerPrefix entries below regardless.
	h.Set("Trailer", "Grpc-Status, Grpc-Message")

	// http.TrailerPrefix tells net/http to flush these as trailers
	// after the body. Safe to set before WriteHeader; net/http reads
	// them when the handler returns.
	h.Set(http.TrailerPrefix+"Grpc-Status", strconv.Itoa(problem.GRPCStatus()))
	h.Set(http.TrailerPrefix+"Grpc-Message", percentEncodeGRPCMessage(detail))

	w.WriteHeader(http.StatusOK)
	return nil
}

// writeConnectUnaryError emits a Connect-unary error: an HTTP status
// from the Connect spec's code→status mapping and a JSON body of the
// form {"code":"...","message":"..."}. Per the Connect spec the
// response Content-Type is always application/json for errors,
// regardless of the request codec.
func writeConnectUnaryError(s *zen.Session, problem publicerr.Problem, detail string) error {
	body := connectError{
		Code:    problem.ConnectCode(),
		Message: detail,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal connect error: %w", err)
	}

	w := s.ResponseWriter()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(problem.ConnectHTTPStatus().Int())
	_, writeErr := w.Write(b)
	return writeErr
}

// writeConnectStreamError emits a Connect streaming error: HTTP 200,
// Content-Type echoes the request subtype (application/connect+json
// or application/connect+proto), body is a single end-stream envelope
// frame carrying the error. End-stream payloads are always JSON
// regardless of the streaming codec, per the Connect spec.
//
// Envelope layout:
//
//	byte  0      flags (0x02 = end-stream)
//	bytes 1..4   payload length, big-endian uint32
//	bytes 5..N   JSON payload
func writeConnectStreamError(s *zen.Session, problem publicerr.Problem, detail string) error {
	payload := connectEndStream{
		Error: &connectError{
			Code:    problem.ConnectCode(),
			Message: detail,
		},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal connect end-stream: %w", err)
	}

	envelope := make([]byte, 5+len(payloadBytes))
	envelope[0] = 0x02 // end-stream flag
	binary.BigEndian.PutUint32(envelope[1:5], uint32(len(payloadBytes)))
	copy(envelope[5:], payloadBytes)

	ct := s.Request().Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/connect+") {
		ct = "application/connect+json"
	}

	w := s.ResponseWriter()
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	_, writeErr := w.Write(envelope)
	return writeErr
}

// connectError is the unary error body and the inner "error" object
// of the streaming end-stream payload.
type connectError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// connectEndStream is the JSON payload of a Connect end-stream
// envelope frame. Error is omitted on success (which we never emit
// from this path).
type connectEndStream struct {
	Error *connectError `json:"error,omitempty"`
}

// percentEncodeGRPCMessage encodes msg per the gRPC HTTP/2 spec for
// the grpc-message trailer: printable ASCII (0x20–0x7E) survives as-is
// except '%' (0x25), which is the escape character. Everything else
// becomes %XX (uppercase hex). Keeps the trailer header-safe across
// proxies and HTTP/1.1 chunked encoders.
func percentEncodeGRPCMessage(msg string) string {
	var b strings.Builder
	b.Grow(len(msg))
	for i := 0; i < len(msg); i++ {
		c := msg[i]
		if c >= 0x20 && c <= 0x7E && c != '%' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
