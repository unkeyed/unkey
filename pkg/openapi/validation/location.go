package validation

import (
	"strings"

	"github.com/pb33f/libopenapi-validator/helpers"
)

// Location prefixes. Every location a caller receives starts with one of these,
// so the caller knows which part of the request to look at.
const (
	locationBody    = "body"
	locationRequest = "request"
)

// locationRedacted stands in for a path segment whose name the caller chose,
// which happens under a free-form object, additionalProperties,
// patternProperties, or propertyNames. Emitting the segment would put a
// caller-chosen name into the response and the request log; emitting nothing
// would hide the depth at which the failure happened.
const locationRedacted = "*"

// renderLocation builds the instance path we report, such as
// "body.variables[1].key" or "query.limit".
//
// instanceLocation is an RFC 6901 pointer into the request ("/variables/1/key")
// and keywordLocation is an RFC 6901 pointer into the schema
// ("/properties/variables/items/properties/key/pattern"). The instance pointer
// alone is not safe to echo: a segment naming a property is a name the caller
// chose unless the schema declares it. Every name we emit is therefore one that
// appears as "properties/<name>" in the schema pointer for the same failure,
// which makes it a token from the specification rather than from the request.
//
// The library also offers a FieldPath on each failure, which is a JSONPath built
// straight from the instance pointer. It is not used, because it carries exactly
// the caller-chosen names this function is here to withhold.
func renderLocation(prefix, instanceLocation, keywordLocation string) string {
	segments := pointerSegments(instanceLocation)
	if len(segments) == 0 {
		return prefix
	}

	declared := pointerSegments(keywordLocation)

	var sb strings.Builder
	sb.WriteString(prefix)
	for _, segment := range segments {
		switch {
		case declaresProperty(declared, segment):
			sb.WriteString(".")
			sb.WriteString(unescapePointerSegment(segment))
		case isArrayIndex(segment):
			sb.WriteString("[")
			sb.WriteString(segment)
			sb.WriteString("]")
		default:
			sb.WriteString(".")
			sb.WriteString(locationRedacted)
		}
	}

	return sb.String()
}

// prefixFor picks the location prefix for one top-level validation error.
func prefixFor(e *validationError) string {
	switch e.ValidationType {
	case helpers.RequestBodyValidation:
		if e.ValidationSubType == helpers.RequestBodyContentType {
			return locationHeader(helpers.ContentTypeHeader)
		}

		return locationBody

	case helpers.ParameterValidation:
		switch e.ValidationSubType {
		case helpers.ParameterValidationPath,
			helpers.ParameterValidationQuery,
			helpers.ParameterValidationHeader,
			helpers.ParameterValidationCookie:
			// The name is the one the specification declares: the validator reached
			// this error by walking the spec's parameter list, not the request's. It
			// still goes through the length guard, so a future code path that fills
			// the field from the request degrades instead of leaking.
			name, ok := specToken(e.ParameterName)
			if !ok {
				return e.ValidationSubType
			}

			return e.ValidationSubType + "." + name
		default:
			return locationRequest
		}

	case helpers.SecurityValidation:
		// An HTTP auth scheme always reads the Authorization header. An apiKey
		// scheme names its own header, query parameter, or cookie, and that name is
		// only available in the message, so those stay at the request level.
		if e.ValidationSubType != "" && e.ValidationSubType != apiKeySubType {
			return locationHeader(helpers.AuthorizationHeader)
		}

		return locationRequest

	default:
		return locationRequest
	}
}

const apiKeySubType = "apiKey"

func locationHeader(name string) string {
	return helpers.ParameterValidationHeader + "." + name
}

// declaresProperty reports whether the schema pointer declares segment as a
// property name.
func declaresProperty(schemaPointer []string, segment string) bool {
	for i := 0; i+1 < len(schemaPointer); i++ {
		if schemaPointer[i] == "properties" && schemaPointer[i+1] == segment {
			return true
		}
	}

	return false
}

func isArrayIndex(segment string) bool {
	if segment == "" {
		return false
	}
	for _, r := range segment {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// pointerSegments splits an RFC 6901 pointer into its escaped segments.
func pointerSegments(pointer string) []string {
	pointer = strings.TrimPrefix(pointer, "/")
	if pointer == "" {
		return nil
	}

	return strings.Split(pointer, "/")
}

func unescapePointerSegment(segment string) string {
	// Order matters: ~01 must decode to ~1, not to /.
	segment = strings.ReplaceAll(segment, "~1", "/")

	return strings.ReplaceAll(segment, "~0", "~")
}
