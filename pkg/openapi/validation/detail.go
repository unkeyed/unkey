package validation

import (
	"regexp"
	"strings"

	"github.com/pb33f/libopenapi-validator/helpers"
)

// genericDetail is the fallback summary, used whenever the validator reports a
// failure whose category this package does not recognise.
const genericDetail = "One or more fields failed validation"

// detailFor builds the top-level summary line.
//
// It is assembled from the validator's structured fields, never from its prose:
// the request method, the spec path template, the parameter's name and location,
// and the failure category. The method and path in the library's own message are
// taken from the request, so a spec with a path parameter that carries a
// credential would put that credential in the summary; SpecPath is the path
// template the request matched, which comes from the specification instead.
func detailFor(e *validationError) string {
	method, methodOK := knownMethod(e.RequestMethod)

	switch e.ValidationType {
	case helpers.RequestBodyValidation:
		if !methodOK || e.SpecPath == "" {
			return genericDetail
		}
		switch {
		case e.ValidationSubType == helpers.RequestBodyContentType:
			return method + " operation for '" + e.SpecPath + "' does not accept the request content type"
		case strings.HasPrefix(e.Reason, reasonBodyEmpty):
			return method + " request body is empty for '" + e.SpecPath + "'"
		default:
			// Byte-identical to the string this API has returned for every rejected
			// body since the validator was introduced, and asserted by about fifty
			// route tests. It says all there is to say at this level now that the
			// errors list carries the specifics, so there is nothing to gain by
			// rewording it.
			return method + " request body for '" + e.SpecPath + "' failed to validate schema"
		}

	case helpers.ParameterValidation:
		entity, predicate, ok := parameterSummary(e)
		if !ok {
			return genericDetail
		}

		return entity + " '" + e.ParameterName + "' " + predicate

	case helpers.SecurityValidation:
		return securityDetail(e)

	case helpers.PathValidation, helpers.RequestValidation:
		if !methodOK {
			return genericDetail
		}

		return method + " request does not match any path and operation in the specification"

	default:
		return genericDetail
	}
}

// reasonBodyEmpty is the library's reason for a required body that was not sent.
const reasonBodyEmpty = "The request body is empty but there is a schema defined"

// reasonBodyUndecodable is the library's reason prefix for a body that is not
// parseable JSON. The rest of that reason is the decoder's error, which can name
// a character from the payload, so it is never forwarded.
const reasonBodyUndecodable = "The request body cannot be decoded"

var knownMethods = map[string]struct{}{
	"GET": {}, "HEAD": {}, "POST": {}, "PUT": {}, "PATCH": {},
	"DELETE": {}, "CONNECT": {}, "OPTIONS": {}, "TRACE": {},
}

// knownMethod bounds the method to the verbs the specification can declare. The
// validator only reports these errors for a request whose method matched an
// operation, so this is belt and braces against a future code path that does
// not.
func knownMethod(method string) (string, bool) {
	if _, ok := knownMethods[method]; !ok {
		return "", false
	}

	return method, true
}

// A parameter failure that never reaches a schema (a missing parameter, a value
// the serialization rules could not decode) is described only by the sentence the
// library writes, and there is no typed kind to read it out of. Those sentences
// interpolate exactly one thing, the parameter's declared name, into an otherwise
// fixed template.
//
// Rather than match the sentence with a regexp and trust that whatever the
// pattern did not capture came from the specification, the template is taken
// apart: the message is split at the quoted name, and both halves have to be in
// the tables below before anything is emitted. That accounts for every byte of
// the message, so a library upgrade that rewords a sentence or interpolates
// something new falls back to fixed text instead of forwarding it. The name
// itself is safe to keep because the library reads it from the specification's
// parameter list, which matters for frontline: it validates customer
// specifications, where a header or query parameter routinely holds a credential,
// and the value must never appear even though the name may.

// parameterEntities are the phrases the library writes before the name. Checked
// against every template in libopenapi-validator v0.13.4.
var parameterEntities = map[string]struct{}{
	"Query parameter":        {},
	"Header parameter":       {},
	"Path parameter":         {},
	"Cookie parameter":       {},
	"Query array parameter":  {},
	"Header array parameter": {},
	"Path array parameter":   {},
	"Cookie array parameter": {},
}

// parameterProblems maps the phrases the library writes after the name onto the
// predicate this package returns. The value is written here, so the mapping is
// also what keeps the message reading as a statement about the location.
var parameterProblems = map[string]string{
	"is missing":                     "is required but was not provided",
	"cannot be empty":                "must not be empty",
	"cannot be decoded":              "could not be decoded",
	"does not match allowed values":  "is not one of the values the schema allows",
	"is not a valid boolean":         "must be a boolean",
	"is not a valid integer":         "must be an integer",
	"is not a valid number":          "must be a number",
	"is not a valid deepObject":      "is not encoded the way the deepObject style requires",
	"is not valid JSON":              "must be valid JSON",
	"delimited incorrectly":          "is not delimited the way the schema requires",
	"is not exploded correctly":      "is not exploded the way the schema requires",
	"value contains reserved values": "contains characters this parameter reserves",
	"contains non-unique items":      "must not contain duplicate items",
	"has too many items":             "contains more items than the schema allows",
	"does not have enough items":     "contains fewer items than the schema requires",
	"failed to validate":             "failed validation",
	"failed schema compilation":      "could not be checked, because the schema declared for it is invalid",
}

// parameterFallback is what a parameter failure says when its message did not
// decompose into a known entity and a known problem.
const parameterFallback = "failed validation"

// parameterSummary rebuilds a parameter failure from its parts: the entity phrase
// and the predicate come from the closed tables above, the name from
// ParameterName.
func parameterSummary(e *validationError) (entity, predicate string, ok bool) {
	name, nameOK := specToken(e.ParameterName)
	if !nameOK {
		return "", "", false
	}

	word := entityWord(e.ValidationSubType)
	if word == "" {
		return "", "", false
	}

	if entity, predicate, split := splitParameterMessage(e.Message, name, word); split {
		return entity, predicate, true
	}

	return word + " parameter", parameterFallback, true
}

// splitParameterMessage decomposes the library's message into the phrase before
// the name and the phrase after it, and reports false unless both are known.
func splitParameterMessage(message, name, word string) (entity, predicate string, ok bool) {
	needle := " '" + name + "' "
	at := strings.Index(message, needle)
	if at < 0 {
		return "", "", false
	}

	entity = message[:at]
	problem := message[at+len(needle):]

	// The entity has to be one of the eight literals and it has to agree with the
	// location the structured fields report, so a message that names a different
	// part of the request than the error does is not trusted either.
	if _, known := parameterEntities[entity]; !known || !strings.HasPrefix(entity, word+" ") {
		return "", "", false
	}

	predicate, known := parameterProblems[problem]
	if !known {
		return "", "", false
	}

	return entity, predicate, true
}

func entityWord(in string) string {
	switch in {
	case helpers.ParameterValidationPath:
		return "Path"
	case helpers.ParameterValidationQuery:
		return "Query"
	case helpers.ParameterValidationHeader:
		return "Header"
	case helpers.ParameterValidationCookie:
		return "Cookie"
	default:
		return ""
	}
}

// securityMessages are the library's security messages. The quoted token in each
// is a security scheme name or an apiKey parameter name, both declared by the
// specification, so the matched message is forwarded as it stands.
var securityMessages = []*regexp.Regexp{
	regexp.MustCompile(`^Security scheme '[^']*' is missing$`),
	regexp.MustCompile(`^Authorization header for '[^']*' scheme$`),
	regexp.MustCompile(`^API Key [^' ]+ not found in (?:header|query|cookies)$`),
}

func securityDetail(e *validationError) string {
	for _, re := range securityMessages {
		if re.MatchString(e.Message) {
			return e.Message
		}
	}

	return "The request is not authorized"
}

// contentTypeFix matches the library's hint for an unsupported content type. The
// count and the list of media types are read from the operation's requestBody,
// so the whole matched hint comes from the specification.
var contentTypeFix = regexp.MustCompile(
	`^The content type is invalid, Use one of the \d+ supported types for this operation: (.+)$`)

// supportedContentTypes pulls the operation's declared media types out of the
// library's hint, returning "" when the hint is not the shape we know.
func supportedContentTypes(howToFix string) string {
	m := contentTypeFix.FindStringSubmatch(howToFix)
	if m == nil || len(m[1]) > maxSpecTokenLen {
		return ""
	}

	return m[1]
}
