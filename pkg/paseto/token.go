package paseto

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	localHeader   = "v4.local."
	publicHeader  = "v4.public."
	tokenOverhead = 64
)

var strictBase64URL = base64.RawURLEncoding.Strict()

type parsedToken struct {
	body   []byte
	footer []byte
}

// UnverifiedFooter extracts a footer without authenticating it. Use the result
// only to select a trusted key. Treat all other footer data as untrusted until
// Decrypt or Verify succeeds.
func UnverifiedFooter(token string) ([]byte, error) {
	header := ""
	switch {
	case strings.HasPrefix(token, localHeader):
		header = localHeader
	case strings.HasPrefix(token, publicHeader):
		header = publicHeader
	default:
		return nil, fmt.Errorf("%w: token does not have a v4 header", ErrInvalidToken)
	}
	parsed, err := parseToken(token, header)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), parsed.footer...), nil
}

func parseToken(token string, expectedHeader string) (parsedToken, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 && len(parts) != 4 {
		return parsedToken{}, fmt.Errorf("%w: token must contain three or four segments", ErrInvalidToken)
	}
	if parts[0]+"."+parts[1]+"." != expectedHeader {
		return parsedToken{}, fmt.Errorf("%w: unexpected token header", ErrInvalidToken)
	}
	if parts[2] == "" {
		return parsedToken{}, fmt.Errorf("%w: token payload is empty", ErrInvalidToken)
	}
	body, err := decodeBase64URLSegment(parts[2], "payload")
	if err != nil {
		return parsedToken{}, err
	}
	if len(body) < tokenOverhead {
		return parsedToken{}, fmt.Errorf("%w: token payload is too short", ErrInvalidToken)
	}

	var footer []byte
	if len(parts) == 4 {
		footer, err = decodeBase64URLSegment(parts[3], "footer")
		if err != nil {
			return parsedToken{}, err
		}
	}
	return parsedToken{body: body, footer: footer}, nil
}

func decodeBase64URLSegment(segment string, name string) ([]byte, error) {
	decoded, err := strictBase64URL.DecodeString(segment)
	if err != nil || strictBase64URL.EncodeToString(decoded) != segment {
		return nil, fmt.Errorf("%w: token %s is not canonical unpadded Base64url", ErrInvalidToken, name)
	}
	return decoded, nil
}

func formatToken(header string, body []byte, footer []byte) string {
	token := header + strictBase64URL.EncodeToString(body)
	if len(footer) != 0 {
		token += "." + strictBase64URL.EncodeToString(footer)
	}
	return token
}
