package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadKey authorizes reading key resources.
//
// Valid resource: urn.Key.
type ReadKey struct{}

func (ReadKey) ActionFor(urn.Key) {}
func (ReadKey) String() string    { return "read_key" }

// EncryptKey authorizes creating recoverable encrypted keys.
//
// Valid resource: urn.Key.
type EncryptKey struct{}

func (EncryptKey) ActionFor(urn.Key) {}
func (EncryptKey) String() string    { return "encrypt_key" }

// DecryptKey authorizes decrypting recoverable key material.
//
// Valid resource: urn.Key.
type DecryptKey struct{}

func (DecryptKey) ActionFor(urn.Key) {}
func (DecryptKey) String() string    { return "decrypt_key" }

// VerifyKey authorizes verifying key resources.
//
// Valid resource: urn.Key.
type VerifyKey struct{}

func (VerifyKey) ActionFor(urn.Key) {}
func (VerifyKey) String() string    { return "verify_key" }
