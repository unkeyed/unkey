package paseto

import (
	"crypto/ed25519"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testClaims struct {
	Claims

	Role string `json:"role,omitempty"`
}

type issuerCollisionClaims struct {
	Claims

	Override string `json:"iss"`
}

type namedEmbeddedClaims struct {
	Claims `json:"claims"`
}

type omittedEmbeddedClaims struct {
	Claims `json:"-"`
}

type customIssuerField struct {
	Override string `json:"iss"`
}

type nestedIssuerClaims struct {
	Claims

	Nested customIssuerField `json:"nested"`
}

type ignoredIssuerClaims struct {
	Claims

	Override string `json:"-"`
}

type customMarshalerClaims struct {
	Claims
}

func (customMarshalerClaims) MarshalJSON() ([]byte, error) {
	return []byte(`{}`), nil
}

type customUnmarshalerClaims struct {
	Claims
}

func (*customUnmarshalerClaims) UnmarshalJSON([]byte) error {
	return nil
}

type unsupportedValueClaims struct {
	Claims

	Values chan string `json:"values"`
}

type methodOnlyClaims struct{}

func (methodOnlyClaims) pasetoClaims() {}

type promotedCustomClaims struct {
	Role string `json:"role"`
}

type pointerEmbeddedClaims struct {
	Claims
	*promotedCustomClaims
}

type unexportedFieldClaims struct {
	Claims

	role string
}

// TestClaims_JSONShape guarantees the registered fields use the names and
// DateTime string type defined by the registered-claims table. The time.Time
// fields use RFC 3339 on the wire. Custom fields remain in the same top-level
// object, and absent claims are omitted.
func TestClaims_JSONShape(t *testing.T) {
	payload := testClaims{
		Claims: Claims{
			Issuer:    "api.unkey.com",
			Subject:   "user_123",
			Audience:  "dashboard",
			ExpiresAt: time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
			NotBefore: time.Time{},
			IssuedAt:  time.Date(2029, 1, 2, 3, 4, 5, 0, time.UTC),
			TokenID:   "token_123",
		},
		Role: "admin",
	}

	encoded, err := encodePayload(payload)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"iss":"api.unkey.com",
		"sub":"user_123",
		"aud":"dashboard",
		"exp":"2030-01-02T03:04:05Z",
		"iat":"2029-01-02T03:04:05Z",
		"jti":"token_123",
		"role":"admin"
	}`, string(encoded))
}

// TestClaimsType_RejectsUnsafeShapes guarantees callers cannot replace a
// registered claim with a custom field or bypass payload checks with custom
// top-level JSON methods.
func TestClaimsType_RejectsUnsafeShapes(t *testing.T) {
	key, err := NewLocalKey(make([]byte, localKeySize))
	require.NoError(t, err)

	t.Run("direct reserved field collision", func(t *testing.T) {
		_, err := NewLocal[issuerCollisionClaims](key)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("named embedded claims", func(t *testing.T) {
		_, err := NewLocal[namedEmbeddedClaims](key)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("omitted embedded claims", func(t *testing.T) {
		_, err := NewLocal[omittedEmbeddedClaims](key)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("anonymous custom fields", func(t *testing.T) {
		_, err := NewLocal[pointerEmbeddedClaims](key)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("duplicate custom JSON name", func(t *testing.T) {
		payloadType := reflect.StructOf([]reflect.StructField{
			{Name: "Claims", Type: claimsType, Anonymous: true},
			{Name: "Role", Type: reflect.TypeFor[string](), Tag: `json:"role"`},
			{Name: "Permission", Type: reflect.TypeFor[string](), Tag: `json:"role"`},
		})
		err := validateClaimsStructType(payloadType)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("duplicate dash JSON name with options", func(t *testing.T) {
		payloadType := reflect.StructOf([]reflect.StructField{
			{Name: "Claims", Type: claimsType, Anonymous: true},
			{Name: "Role", Type: reflect.TypeFor[string](), Tag: `json:"-,omitempty"`},
			{Name: "Permission", Type: reflect.TypeFor[string](), Tag: `json:"-,omitzero"`},
		})
		err := validateClaimsStructType(payloadType)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("invalid JSON name falls back to field name", func(t *testing.T) {
		payloadType := reflect.StructOf([]reflect.StructField{
			{Name: "Claims", Type: claimsType, Anonymous: true},
			{Name: "Role", Type: reflect.TypeFor[string](), Tag: `json:"💥"`},
			{Name: "Permission", Type: reflect.TypeFor[string](), Tag: `json:"Role"`},
		})
		err := validateClaimsStructType(payloadType)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("pointer payload", func(t *testing.T) {
		_, err := NewLocal[*testClaims](key)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("custom marshaler", func(t *testing.T) {
		_, err := NewLocal[customMarshalerClaims](key)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("custom unmarshaler", func(t *testing.T) {
		_, err := NewLocal[customUnmarshalerClaims](key)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})

	t.Run("claim-set method without embedded claims", func(t *testing.T) {
		_, err := NewLocal[methodOnlyClaims](key)
		require.ErrorIs(t, err, ErrInvalidClaims)
	})
}

// TestClaimsType_AllowsReservedNamesOutsideTopLevel guarantees the registered
// names remain available in nested objects and ignored Go fields. The PASETO
// reservation applies only to the top-level JSON object.
func TestClaimsType_AllowsReservedNamesOutsideTopLevel(t *testing.T) {
	key, err := NewLocalKey(make([]byte, localKeySize))
	require.NoError(t, err)
	local, err := NewLocal[nestedIssuerClaims](key)
	require.NoError(t, err)
	message := Message[nestedIssuerClaims]{
		Payload: nestedIssuerClaims{
			Claims: Claims{Issuer: "top-level"},
			Nested: customIssuerField{Override: "nested"},
		},
		Footer: nil,
	}
	token, err := local.Encrypt(message)
	require.NoError(t, err)
	decrypted, err := local.Decrypt(token)
	require.NoError(t, err)
	require.Equal(t, message, decrypted)

	_, err = NewLocal[ignoredIssuerClaims](key)
	require.NoError(t, err)
	_, err = NewLocal[unexportedFieldClaims](key)
	require.NoError(t, err)
}

// TestClaimsEncoding_RejectsUnsupportedCustomValues guarantees JSON encoding
// errors fail token creation instead of producing a partial payload.
func TestClaimsEncoding_RejectsUnsupportedCustomValues(t *testing.T) {
	key, err := NewLocalKey(make([]byte, localKeySize))
	require.NoError(t, err)
	local, err := NewLocal[unsupportedValueClaims](key)
	require.NoError(t, err)
	_, err = local.Encrypt(Message[unsupportedValueClaims]{
		Payload: unsupportedValueClaims{
			Claims: Claims{},
			Values: make(chan string),
		},
		Footer: nil,
	})
	require.ErrorIs(t, err, ErrInvalidClaims)

	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	signer, err := NewSigner[unsupportedValueClaims](privateKey)
	require.NoError(t, err)
	_, err = signer.Sign(Message[unsupportedValueClaims]{
		Payload: unsupportedValueClaims{
			Claims: Claims{},
			Values: make(chan string),
		},
		Footer: nil,
	})
	require.ErrorIs(t, err, ErrInvalidClaims)
}
