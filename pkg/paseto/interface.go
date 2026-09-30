package paseto

// Message contains the typed payload and authenticated footer of a PASETO.
// The footer is authenticated but is never encrypted.
type Message[T ClaimSet] struct {
	Payload T
	Footer  []byte
}

// Encrypter creates authenticated v4.local tokens.
// Implementations are safe for concurrent use.
type Encrypter[T ClaimSet] interface {
	Encrypt(message Message[T]) (string, error)
}

// Decrypter authenticates and decrypts v4.local tokens.
// Implementations are safe for concurrent use.
type Decrypter[T ClaimSet] interface {
	Decrypt(token string) (Message[T], error)
}

// Signer creates authenticated v4.public tokens.
// Implementations are safe for concurrent use.
type Signer[T ClaimSet] interface {
	Sign(message Message[T]) (string, error)
}

// Verifier authenticates v4.public tokens.
// Implementations are safe for concurrent use.
type Verifier[T ClaimSet] interface {
	Verify(token string) (Message[T], error)
}
