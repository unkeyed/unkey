package paseto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"time"
	"unicode/utf8"
)

var rfc3339DateTimePattern = regexp.MustCompile(
	`^[0-9]{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12][0-9]|3[01])T(?:[01][0-9]|2[0-3]):[0-5][0-9]:(?:[0-5][0-9]|60)(?:[.][0-9]+)?(?:Z|[+-](?:[01][0-9]|2[0-3]):[0-5][0-9])$`,
)

type payloadObject struct {
	claims Claims
	fields map[string]json.RawMessage
}

func encodePayload[T ClaimSet](payload T) ([]byte, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: encode payload: %v", ErrInvalidClaims, err)
	}
	if _, err := inspectPayload(encoded); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidClaims, err)
	}
	return encoded, nil
}

func decodePayload[T ClaimSet](encoded []byte) (T, error) {
	var payload T
	object, err := inspectPayload(encoded)
	if err != nil {
		return payload, err
	}
	for name := range registeredClaimNames {
		delete(object.fields, name)
	}
	// inspectPayload produced every RawMessage from valid JSON, so this map
	// cannot contain a value that json.Marshal rejects.
	customClaims, _ := json.Marshal(object.fields)
	if err := json.Unmarshal(customClaims, &payload); err != nil {
		return payload, fmt.Errorf("decode payload: %w", err)
	}
	if err := setRegisteredClaims(&payload, object.claims); err != nil {
		return payload, err
	}
	return payload, nil
}

func inspectPayload(encoded []byte) (payloadObject, error) {
	if !utf8.Valid(encoded) {
		return payloadObject{}, fmt.Errorf("payload is not valid UTF-8")
	}
	object := map[string]json.RawMessage{}
	if err := json.Unmarshal(encoded, &object); err != nil {
		return payloadObject{}, fmt.Errorf("decode payload object: %w", err)
	}
	if err := requireUniqueJSONObject(encoded); err != nil {
		return payloadObject{}, err
	}
	claims, err := parseRegisteredClaims(object)
	if err != nil {
		return payloadObject{}, err
	}
	return payloadObject{claims: claims, fields: object}, nil
}

func requireUniqueJSONObject(encoded []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	delimiter, ok := first.(json.Delim)
	if !ok || delimiter != '{' {
		return fmt.Errorf("payload must be a JSON object")
	}
	if err := inspectJSONObject(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("payload contains data after the JSON object")
		}
		return fmt.Errorf("decode payload: %w", err)
	}
	return nil
}

func inspectJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode JSON value: %w", err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if delimiter == '{' {
		return inspectJSONObject(decoder)
	}
	// At a value boundary, the decoder can return only an opening object or
	// array delimiter. The object case returned above, so this is an array.
	for decoder.More() {
		if err := inspectJSONValue(decoder); err != nil {
			return err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("decode JSON array: %w", err)
	}
	return nil
}

func inspectJSONObject(decoder *json.Decoder) error {
	seen := map[string]struct{}{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode JSON object key: %w", err)
		}
		// A decoder inside an object returns a string token for every key.
		key := token.(string)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("payload contains duplicate key %q", key)
		}
		seen[key] = struct{}{}
		if err := inspectJSONValue(decoder); err != nil {
			return err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("decode JSON object: %w", err)
	}
	return nil
}

func parseRegisteredClaims(object map[string]json.RawMessage) (Claims, error) {
	var claims Claims
	var err error
	if claims.Issuer, err = parseStringClaim(object, "iss"); err != nil {
		return Claims{}, err
	}
	if claims.Subject, err = parseStringClaim(object, "sub"); err != nil {
		return Claims{}, err
	}
	if claims.Audience, err = parseStringClaim(object, "aud"); err != nil {
		return Claims{}, err
	}
	if claims.ExpiresAt, err = parseTimeClaim(object, "exp"); err != nil {
		return Claims{}, err
	}
	if claims.NotBefore, err = parseTimeClaim(object, "nbf"); err != nil {
		return Claims{}, err
	}
	if claims.IssuedAt, err = parseTimeClaim(object, "iat"); err != nil {
		return Claims{}, err
	}
	if claims.TokenID, err = parseStringClaim(object, "jti"); err != nil {
		return Claims{}, err
	}
	return claims, nil
}

func parseStringClaim(object map[string]json.RawMessage, name string) (string, error) {
	raw, exists := object[name]
	if !exists {
		return "", nil
	}
	if len(raw) == 0 || raw[0] != '"' {
		return "", fmt.Errorf("registered claim %q must be a string", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("decode registered claim %q: %w", name, err)
	}
	return value, nil
}

func parseTimeClaim(object map[string]json.RawMessage, name string) (time.Time, error) {
	if _, exists := object[name]; !exists {
		return time.Time{}, nil
	}
	value, err := parseStringClaim(object, name)
	if err != nil {
		return time.Time{}, err
	}
	// time.Parse accepts some forms outside RFC 3339, such as a comma before
	// fractional seconds. Check the exact syntax before parsing the value.
	if !rfc3339DateTimePattern.MatchString(value) {
		return time.Time{}, fmt.Errorf("registered claim %q must use RFC 3339", name)
	}
	valueToParse := value
	leapSecond := value[17:19] == "60"
	if leapSecond {
		// RFC 3339 permits leap seconds, but time.Parse does not. Parse the prior
		// second, then advance to the same instant on Go's time scale.
		valueToParse = value[:17] + "59" + value[19:]
	}
	parsed, err := time.Parse(time.RFC3339, valueToParse)
	if err != nil {
		return time.Time{}, fmt.Errorf("registered claim %q must use RFC 3339: %w", name, err)
	}
	if leapSecond {
		normalized := parsed.Add(time.Second)
		utc := normalized.UTC()
		atLeapSecondBoundary := utc.Day() == 1 &&
			(utc.Month() == time.January || utc.Month() == time.July) &&
			utc.Hour() == 0 && utc.Minute() == 0 && utc.Second() == 0
		if !atLeapSecondBoundary {
			return time.Time{}, fmt.Errorf("registered claim %q contains an invalid leap second", name)
		}
		parsed = normalized
	}
	if parsed.IsZero() {
		return time.Time{}, fmt.Errorf("registered claim %q cannot equal the absent zero value", name)
	}
	return parsed, nil
}

func setRegisteredClaims[T ClaimSet](payload *T, claims Claims) error {
	value := reflect.ValueOf(payload).Elem()
	if value.Type() == claimsType {
		value.Set(reflect.ValueOf(claims))
		return nil
	}
	for index := range value.NumField() {
		field := value.Type().Field(index)
		if field.Anonymous && field.Type == claimsType {
			value.Field(index).Set(reflect.ValueOf(claims))
			return nil
		}
	}
	return fmt.Errorf("%w: payload type does not embed Claims", ErrInvalidClaims)
}
