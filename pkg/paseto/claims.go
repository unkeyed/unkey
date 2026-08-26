package paseto

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
)

var (
	claimsType          = reflect.TypeFor[Claims]()
	jsonMarshalerType   = reflect.TypeFor[json.Marshaler]()
	jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()
)

var registeredClaimNames = map[string]struct{}{
	"aud": {},
	"exp": {},
	"iat": {},
	"iss": {},
	"jti": {},
	"nbf": {},
	"sub": {},
}

// Claims contains the claims reserved by the PASETO specification. All claims
// are optional. Embed Claims in an application payload type. Define custom
// claims as exported fields so encoding/json can encode them.
type Claims struct {
	Issuer    string    `json:"iss,omitempty"`
	Subject   string    `json:"sub,omitempty"`
	Audience  string    `json:"aud,omitempty"`
	ExpiresAt time.Time `json:"exp,omitzero"`
	NotBefore time.Time `json:"nbf,omitzero"`
	IssuedAt  time.Time `json:"iat,omitzero"`
	TokenID   string    `json:"jti,omitempty"`
}

func (Claims) pasetoClaims() {}

// ClaimSet is implemented by payload types that embed [Claims]. Its private
// method prevents payload types from satisfying this interface by accident.
type ClaimSet interface {
	pasetoClaims()
}

func validateClaimsType[T ClaimSet]() error {
	return validateClaimsStructType(reflect.TypeFor[T]())
}

func validateClaimsStructType(payloadType reflect.Type) error {
	if payloadType == claimsType {
		return nil
	}
	if payloadType.Kind() != reflect.Struct {
		return fmt.Errorf("%w: payload type must be a struct", ErrInvalidClaims)
	}
	if payloadType.Implements(jsonMarshalerType) || reflect.PointerTo(payloadType).Implements(jsonMarshalerType) {
		return fmt.Errorf("%w: payload type must not implement json.Marshaler", ErrInvalidClaims)
	}
	if payloadType.Implements(jsonUnmarshalerType) || reflect.PointerTo(payloadType).Implements(jsonUnmarshalerType) {
		return fmt.Errorf("%w: payload type must not implement json.Unmarshaler", ErrInvalidClaims)
	}

	claimsFields := 0
	customNames := map[string]struct{}{}
	for index := range payloadType.NumField() {
		field := payloadType.Field(index)
		if field.Anonymous && field.Type == claimsType {
			_, tagged, ignored := jsonFieldName(field)
			if tagged || ignored {
				return fmt.Errorf("%w: embedded Claims must not use a JSON name or %q", ErrInvalidClaims, "-")
			}
			claimsFields++
			continue
		}
		if field.Anonymous {
			return fmt.Errorf("%w: only Claims may be embedded", ErrInvalidClaims)
		}
		if !field.IsExported() {
			continue
		}
		name, _, ignored := jsonFieldName(field)
		if ignored {
			continue
		}
		if _, reserved := registeredClaimNames[name]; reserved {
			return fmt.Errorf("%w: custom field %s uses reserved claim %q", ErrInvalidClaims, field.Name, name)
		}
		if _, duplicate := customNames[name]; duplicate {
			return fmt.Errorf("%w: multiple custom fields use JSON name %q", ErrInvalidClaims, name)
		}
		customNames[name] = struct{}{}
	}
	if claimsFields != 1 {
		return fmt.Errorf("%w: payload type must embed Claims exactly once", ErrInvalidClaims)
	}
	return nil
}

func jsonFieldName(field reflect.StructField) (name string, tagged bool, ignored bool) {
	tag := field.Tag.Get("json")
	name, _, _ = strings.Cut(tag, ",")
	if tag == "-" {
		return "", false, true
	}
	if name != "" && validJSONTagName(name) {
		return name, true, false
	}
	return field.Name, false, false
}

func validJSONTagName(name string) bool {
	for _, character := range name {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", character):
		case !unicode.IsLetter(character) && !unicode.IsDigit(character):
			return false
		}
	}
	return true
}
