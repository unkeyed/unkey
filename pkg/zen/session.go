package zen

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Session encapsulates the state and utilities for handling a single HTTP request.
// It wraps the standard http.ResponseWriter and http.Request with additional
// functionality for parsing requests and generating responses.
//
// Sessions are pooled and reused between requests to reduce memory allocations.
// References to sessions, requests, or responses should not be stored beyond
// the handler's execution.
//
// A new Session is created for each request and passed to the route handler.
// The Session is automatically reset and returned to the pool after the request
// is handled.
type Session struct {
	requestID string

	w http.ResponseWriter // Wrapped with statusRecorder to capture status code
	r *http.Request

	// The workspace making the request.
	// We extract this from the root key or regular key
	// and must set it before the metrics middleware finishes.
	WorkspaceID string

	requestBody    []byte
	responseStatus int
	responseBody   []byte

	// ClickHouse request logging control - defaults to true (log by default)
	logRequestToClickHouse bool

	// internalError stores the internal error message for logging to ClickHouse.
	// This is set by the error handling middleware before it converts the error
	// to an HTTP response, allowing the metrics middleware to log the full error.
	internalError string
}

func (s *Session) Init(w http.ResponseWriter, r *http.Request, maxBodySize int64) error {
	s.requestID = uid.New(uid.RequestPrefix)

	// Wrap ResponseWriter with status recorder
	s.w = &statusRecorder{
		ResponseWriter: w,
		statusCode:     0, // Default to 0, this should always be overwritten by the metrics middleware
		written:        false,
	}

	s.r = r
	s.logRequestToClickHouse = true // Default to logging requests to ClickHouse

	// Apply body size limit if configured
	// Note: MaxBytesReader needs the original unwrapped ResponseWriter, so we pass w directly
	if maxBodySize > 0 {
		s.r.Body = http.MaxBytesReader(w, s.r.Body, maxBodySize)
	}

	// Read and cache the request body so metrics middleware can access it even on early errors.
	// We need to replace r.Body with a fresh reader afterwards so other middleware
	// can still read the body if necessary.
	var err error
	s.requestBody, err = io.ReadAll(s.r.Body)
	closeErr := s.r.Body.Close()

	// Handle read errors (including MaxBytesError)
	if err != nil {
		// Check if this is a MaxBytesError from http.MaxBytesReader
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return fault.Wrap(err,
				fault.Code(codes.User.BadRequest.RequestBodyTooLarge.URN()),
				fault.Internal(fmt.Sprintf("request body exceeds size limit of %d bytes", maxBytesErr.Limit)),
				fault.Public(fmt.Sprintf("The request body exceeds the maximum allowed size of %d bytes.", maxBytesErr.Limit)),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.User.BadRequest.RequestBodyUnreadable.URN()),
			fault.Internal("unable to read request body"),
			fault.Public("The request body could not be read."),
		)
	}

	// Handle close error (incase that ever happens)
	if closeErr != nil {
		return fault.Wrap(closeErr,
			fault.Internal("failed to close request body"),
			fault.Public("An error occurred processing the request."),
		)
	}

	// Replace body with a fresh reader for subsequent middleware
	s.r.Body = io.NopCloser(bytes.NewReader(s.requestBody))
	s.WorkspaceID = ""
	return nil
}

// AuthorizedWorkspaceID returns the workspace ID associated with the authenticated
// request. This is populated by authentication middleware.
//
// Returns an empty string if no authenticated workspace ID is available.
func (s *Session) AuthorizedWorkspaceID() string {
	return s.WorkspaceID
}

// DisableClickHouseLogging prevents this request from being logged to ClickHouse.
// By default, all requests are logged to ClickHouse unless explicitly disabled.
//
// This is useful for internal endpoints like health checks, OpenAPI specs,
// or requests that should not appear in analytics.
func (s *Session) DisableClickHouseLogging() {
	s.logRequestToClickHouse = false
}

// ShouldLogRequestToClickHouse returns whether this request should be logged to ClickHouse.
// Returns true by default, false only if explicitly disabled.
func (s *Session) ShouldLogRequestToClickHouse() bool {
	return s.logRequestToClickHouse
}

// SetInternalError stores the internal error message for logging purposes.
// This should be called by error handling middleware before converting
// errors to HTTP responses.
func (s *Session) SetInternalError(err string) {
	s.internalError = err
}

// InternalError returns the stored internal error message for logging.
func (s *Session) InternalError() string {
	return s.internalError
}

func (s *Session) UserAgent() string {
	return s.r.UserAgent()
}

// Location returns the client's IP address, checking X-Forwarded-For header first,
// then falling back to RemoteAddr. Ports are stripped from the returned IP.
func (s *Session) Location() string {
	xff := s.r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				return stripPort(ip)
			}
		}
	}

	// Fall back to RemoteAddr
	return stripPort(s.r.RemoteAddr)
}

// stripPort removes the port from an address string.
// Handles IPv4 (192.168.1.1:8080), IPv6 with brackets ([::1]:8080), and plain addresses.
func stripPort(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}
	// No port present or invalid format, return as-is
	return addr
}

// Request returns the underlying http.Request.
// This allows direct access to the standard library request features.
//
// Note: The returned request should not be stored across requests or modified
// after the handler returns.
func (s *Session) Request() *http.Request {
	return s.r
}

// RequestID returns the request id for this session.
func (s *Session) RequestID() string {
	return s.requestID
}

// ResponseWriter returns the http.ResponseWriter with status code capturing.
// This allows direct access to the standard library response features.
//
// Direct manipulation of the ResponseWriter should be avoided when possible
// in favor of using the Session's response methods like JSON or Send.
func (s *Session) ResponseWriter() http.ResponseWriter {
	return s.w
}

// StatusCode returns the HTTP status code that was written to the response.
// Returns 200 if no status code has been explicitly set.
func (s *Session) StatusCode() int {
	if recorder, ok := s.w.(*statusRecorder); ok {
		return recorder.statusCode
	}

	return 200
}

// BindBody parses the request body as JSON into the provided destination struct.
// The destination must be a pointer to a struct.
//
// If parsing fails, an appropriate error is returned. The original request body is
// stored in the session for potential reuse or logging.
//
// Example:
//
//	var user User
//	if err := sess.BindBody(&user); err != nil {
//	    return err
//	}
//	// Use the parsed user data
func (s *Session) BindBody(dst any) error {
	err := json.Unmarshal(s.requestBody, dst)
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("failed to unmarshal request body"),
			fault.Public("The request body was not valid JSON."),
		)
	}

	return nil
}

// BindQuery parses URL query parameters into the provided destination struct.
// The destination must be a pointer to a struct with json tags that match
// the query parameter names.
//
// Example:
//
//	var params struct {
//	    Limit  int    `json:"limit"`
//	    Cursor string `json:"cursor"`
//	    Filter string `json:"filter"`
//	}
//	if err := sess.BindQuery(&params); err != nil {
//	    return err
//	}
//	// Use params.Limit, params.Cursor, and params.Filter
func (s *Session) BindQuery(dst interface{}) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fault.New("destination must be a non-nil pointer")
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fault.New("destination must be a pointer to a struct")
	}

	typ := elem.Type()
	query := s.r.URL.Query()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Split the json tag to handle options like omitempty
		parts := strings.Split(jsonTag, ",")
		name := parts[0]

		// Check if the query parameter exists
		values, exists := query[name]
		if !exists || len(values) == 0 {
			continue
		}

		fieldValue := elem.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		// nolint:exhaustive
		switch fieldValue.Kind() {
		case reflect.String:
			{
				fieldValue.SetString(values[0])
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			{
				intVal, err := strconv.ParseInt(values[0], 10, 64)
				if err != nil {
					return fault.Wrap(err,
						fault.Internal(fmt.Sprintf("could not parse %s as integer", name)),
						fault.Public(fmt.Sprintf("The query parameter '%s' must be a valid integer.", name)))
				}
				fieldValue.SetInt(intVal)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			{
				uintVal, err := strconv.ParseUint(values[0], 10, 64)
				if err != nil {
					return fault.Wrap(err,
						fault.Internal(fmt.Sprintf("could not parse %s as unsigned integer", name)),
						fault.Public(fmt.Sprintf("The query parameter '%s' must be a valid unsigned integer.", name)))
				}
				fieldValue.SetUint(uintVal)
			}
		case reflect.Float32, reflect.Float64:
			{
				floatVal, err := strconv.ParseFloat(values[0], 64)
				if err != nil {
					return fault.Wrap(err,
						fault.Internal(fmt.Sprintf("could not parse %s as float", name)),
						fault.Public(fmt.Sprintf("The query parameter '%s' must be a valid floating point number.", name)))
				}
				fieldValue.SetFloat(floatVal)
			}
		case reflect.Bool:
			{
				boolVal, err := strconv.ParseBool(values[0])
				if err != nil {
					return fault.Wrap(err,
						fault.Internal(fmt.Sprintf("could not parse %s as boolean", name)),
						fault.Public(fmt.Sprintf("The query parameter '%s' must be a valid boolean value (true/false).", name)))
				}
				fieldValue.SetBool(boolVal)
			}
		case reflect.Slice:
			{
				// Handle slices differently based on element type
				sliceType := fieldValue.Type().Elem().Kind()
				slice := reflect.MakeSlice(fieldValue.Type(), len(values), len(values))

				for j, val := range values {
					// nolint:exhaustive
					switch sliceType {
					case reflect.String:
						{
							slice.Index(j).SetString(val)
						}
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						{
							intVal, err := strconv.ParseInt(val, 10, 64)
							if err != nil {
								return fault.Wrap(err,
									fault.Internal(fmt.Sprintf("could not parse item in %s as integer", name)),
									fault.Public(fmt.Sprintf("All items in query parameter '%s' must be valid integers.", name)))
							}
							slice.Index(j).SetInt(intVal)
						}
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						{
							uintVal, err := strconv.ParseUint(val, 10, 64)
							if err != nil {
								return fault.Wrap(err,
									fault.Internal(fmt.Sprintf("could not parse item in %s as unsigned integer", name)),
									fault.Public(fmt.Sprintf("All items in query parameter '%s' must be valid unsigned integers.", name)))
							}
							slice.Index(j).SetUint(uintVal)
						}
					case reflect.Float32, reflect.Float64:
						{
							floatVal, err := strconv.ParseFloat(val, 64)
							if err != nil {
								return fault.Wrap(err,
									fault.Internal(fmt.Sprintf("could not parse item in %s as float", name)),
									fault.Public(fmt.Sprintf("All items in query parameter '%s' must be valid floating point numbers.", name)))
							}
							slice.Index(j).SetFloat(floatVal)
						}
					case reflect.Bool:
						{
							boolVal, err := strconv.ParseBool(val)
							if err != nil {
								return fault.Wrap(err,
									fault.Internal(fmt.Sprintf("could not parse item in %s as boolean", name)),
									fault.Public(fmt.Sprintf("All items in query parameter '%s' must be valid boolean values (true/false).", name)))
							}
							slice.Index(j).SetBool(boolVal)
						}
					default:
						return fault.New(fmt.Sprintf("unsupported slice element type for field %s", name),
							fault.Internal("type conversion error"), fault.Public(fmt.Sprintf("The query parameter '%s' contains elements of an unsupported type.", name)))
					}
				}
				fieldValue.Set(slice)
			}
		default:
			return fault.New(fmt.Sprintf("unsupported field type for %s", name),
				fault.Internal("type conversion error"), fault.Public(fmt.Sprintf("The query parameter '%s' is of an unsupported type.", name)))
		}
	}

	return nil
}

// AddHeader adds a key-value pair to the response headers.
// This method can be called multiple times with the same key to add
// multiple values for the same header.
func (s *Session) AddHeader(key, val string) {
	s.w.Header().Add(key, val)
}

func (s *Session) send(status int, body []byte) error {
	// Store the status and body for middleware use
	// Unlike the headers, we can't access it on the responseWriter
	s.responseStatus = status
	s.responseBody = body

	s.w.WriteHeader(status)
	_, err := s.w.Write(body)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to send bytes"), fault.Public("Unable to send response body."))
	}

	return nil
}

// JSON sets the response status code and sends a JSON-encoded response.
// It automatically sets the Content-Type header to application/json.
//
// The body is marshaled using github.com/bytedance/sonic
// If marshaling fails, an error is returned.
//
// Example:
//
//	return sess.JSON(http.StatusOK, map[string]interface{}{
//	    "user": user,
//	    "token": token,
//	})
func (s *Session) JSON(status int, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Internal("json marshal failed"),
			fault.Public("The response body could not be marshalled to JSON."),
		)
	}

	s.ResponseWriter().Header().Add("Content-Type", "application/json")
	return s.send(status, b)
}

// HTML sends an HTML response with the given status code.
func (s *Session) HTML(status int, body []byte) error {
	s.w.Header().Set("Content-Type", "text/html")
	return s.send(status, body)
}

// Plain sends a plain text response with the given status code.
func (s *Session) Plain(status int, body []byte) error {
	s.w.Header().Set("Content-Type", "text/plain")
	return s.send(status, body)
}

// Send sets the response status code and sends raw bytes as the response body.
// This method is useful for non-JSON responses like binary data or plain text.
//
// Unlike [JSON], this method does not set any Content-Type header automatically.
func (s *Session) Send(status int, body []byte) error {
	return s.send(status, body)
}

// reset is called automatically before the session is returned to the pool.
// It resets all fields to their null value to prevent leaking data between
// requests.
func (s *Session) reset() {
	s.requestID = ""

	s.w = nil
	s.r = nil

	s.requestBody = nil
	s.responseStatus = 0
	s.responseBody = nil
	s.logRequestToClickHouse = true // Reset ClickHouse logging control to default (enabled)
	s.internalError = ""
}
