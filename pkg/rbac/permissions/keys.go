package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadKey authorizes reading a key.
type ReadKey struct{}

func (ReadKey) ActionFor(urn.Key) {}
func (ReadKey) String() string    { return "read_key" }

// WriteKey authorizes creating or updating a key and its role or permission assignments.
type WriteKey struct{}

func (WriteKey) ActionFor(urn.Key) {}
func (WriteKey) String() string    { return "write_key" }

// DeleteKey authorizes deleting a key.
type DeleteKey struct{}

func (DeleteKey) ActionFor(urn.Key) {}
func (DeleteKey) String() string    { return "delete_key" }

// DecryptKey authorizes decrypting recoverable key material.
type DecryptKey struct{}

func (DecryptKey) ActionFor(urn.Key) {}
func (DecryptKey) String() string    { return "decrypt_key" }

// VerifyKey authorizes verifying a key.
type VerifyKey struct{}

func (VerifyKey) ActionFor(urn.Key) {}
func (VerifyKey) String() string    { return "verify_key" }
