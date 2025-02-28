package zen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Session is a thin wrapper on top of go's standard library net/http
// It offers convenience methods to parse requests and send responses.
//
// Session structs are reused to ease the load on the GC.
// All references to sessions, request bodies or anything within must not be
// used outside of the handler. Make a copy of them if you need to.
type Session struct {
	ctx       context.Context
	requestID string

	w http.ResponseWriter
	r *http.Request

	// The workspace making the request.
	// We extract this from the root key or regular key
	// and must set it before the metrics middleware finishes.
	workspaceID string

	requestBody    []byte
	responseStatus int
	responseBody   []byte
}

func (s *Session) init(w http.ResponseWriter, r *http.Request) error {
	s.ctx = r.Context()
	s.requestID = uid.Request()
	s.w = w
	s.r = r

	s.workspaceID = ""
	return nil
}

func (s *Session) Context() context.Context {
	return s.ctx

}

// AuthorizedWorkspaceID returns the workspaceID of the root key used as authentication mechanism.
//
// If the `WithRootKeyAuth` middleware is used, it is guaranteed to be populated.
// The request would've aborted and returned early if authentication failed.
// Otherwise an empty string is returned.
func (s *Session) AuthorizedWorkspaceID() string {
	return s.workspaceID
}

// Request returns the underlying http.Request.
//
// Do not store references or modify it outside of the handler function.
func (s *Session) Request() *http.Request {
	return s.r
}

func (s *Session) ResponseWriter() http.ResponseWriter {
	return s.w
}

func (s *Session) BindBody(dst any) error {
	var err error
	s.requestBody, err = io.ReadAll(s.r.Body)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("unable to read request body", "The request body is malformed."))
	}
	defer s.r.Body.Close()

	err = json.Unmarshal(s.requestBody, dst)
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("failed to unmarshal request body", "The request body was not valid json."),
		)
	}
	return nil
}

// BindQuery binds query parameters to a struct.
// The struct should have json tags which will be used to match query parameter names.
// For example, a field with tag `json:"limit"` will match the query parameter "limit".
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

		// Handle different field types
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
						fault.WithDesc(fmt.Sprintf("could not parse %s as integer", name),
							fmt.Sprintf("The query parameter '%s' must be a valid integer.", name)))
				}
				fieldValue.SetInt(intVal)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			{
				uintVal, err := strconv.ParseUint(values[0], 10, 64)
				if err != nil {
					return fault.Wrap(err,
						fault.WithDesc(fmt.Sprintf("could not parse %s as unsigned integer", name),
							fmt.Sprintf("The query parameter '%s' must be a valid unsigned integer.", name)))
				}
				fieldValue.SetUint(uintVal)
			}
		case reflect.Float32, reflect.Float64:
			{
				floatVal, err := strconv.ParseFloat(values[0], 64)
				if err != nil {
					return fault.Wrap(err,
						fault.WithDesc(fmt.Sprintf("could not parse %s as float", name),
							fmt.Sprintf("The query parameter '%s' must be a valid floating point number.", name)))
				}
				fieldValue.SetFloat(floatVal)
			}
		case reflect.Bool:
			{
				boolVal, err := strconv.ParseBool(values[0])
				if err != nil {
					return fault.Wrap(err,
						fault.WithDesc(fmt.Sprintf("could not parse %s as boolean", name),
							fmt.Sprintf("The query parameter '%s' must be a valid boolean value (true/false).", name)))
				}
				fieldValue.SetBool(boolVal)
			}
		case reflect.Slice:
			{
				// Handle slices differently based on element type
				sliceType := fieldValue.Type().Elem().Kind()
				slice := reflect.MakeSlice(fieldValue.Type(), len(values), len(values))

				for j, val := range values {
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
									fault.WithDesc(fmt.Sprintf("could not parse item in %s as integer", name),
										fmt.Sprintf("All items in query parameter '%s' must be valid integers.", name)))
							}
							slice.Index(j).SetInt(intVal)
						}
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						{
							uintVal, err := strconv.ParseUint(val, 10, 64)
							if err != nil {
								return fault.Wrap(err,
									fault.WithDesc(fmt.Sprintf("could not parse item in %s as unsigned integer", name),
										fmt.Sprintf("All items in query parameter '%s' must be valid unsigned integers.", name)))
							}
							slice.Index(j).SetUint(uintVal)
						}
					case reflect.Float32, reflect.Float64:
						{
							floatVal, err := strconv.ParseFloat(val, 64)
							if err != nil {
								return fault.Wrap(err,
									fault.WithDesc(fmt.Sprintf("could not parse item in %s as float", name),
										fmt.Sprintf("All items in query parameter '%s' must be valid floating point numbers.", name)))
							}
							slice.Index(j).SetFloat(floatVal)
						}
					case reflect.Bool:
						{
							boolVal, err := strconv.ParseBool(val)
							if err != nil {
								return fault.Wrap(err,
									fault.WithDesc(fmt.Sprintf("could not parse item in %s as boolean", name),
										fmt.Sprintf("All items in query parameter '%s' must be valid boolean values (true/false).", name)))
							}
							slice.Index(j).SetBool(boolVal)
						}
					default:
						return fault.New(fmt.Sprintf("unsupported slice element type for field %s", name),
							fault.WithDesc("type conversion error",
								fmt.Sprintf("The query parameter '%s' contains elements of an unsupported type.", name)))
					}
				}
				fieldValue.Set(slice)
			}
		default:
			return fault.New(fmt.Sprintf("unsupported field type for %s", name),
				fault.WithDesc("type conversion error",
					fmt.Sprintf("The query parameter '%s' is of an unsupported type.", name)))
		}
	}

	return nil
}

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
		return fault.Wrap(err, fault.WithDesc("failed to send bytes", "Unable to send response body."))
	}

	return nil
}

// Send sets the response status and header
// It then marshals the body as JSON and sends it to the client.
func (s *Session) JSON(status int, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("json marshal failed", "The response body could not be marshalled to JSON."),
		)
	}
	s.ResponseWriter().Header().Add("Content-Type", "application/json")
	return s.send(status, b)
}
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
}
