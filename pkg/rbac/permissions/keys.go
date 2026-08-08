package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateKey authorizes creating a key resource.
//
// Valid resource: urn.Key.
type CreateKey struct{}

func (CreateKey) ActionFor(urn.Key) {}
func (CreateKey) String() string    { return "create_key" }

// ReadKey authorizes reading key resources.
//
// Valid resource: urn.Key.
type ReadKey struct{}

func (ReadKey) ActionFor(urn.Key) {}
func (ReadKey) String() string    { return "read_key" }

// UpdateKey authorizes updating key resources.
//
// Valid resource: urn.Key.
type UpdateKey struct{}

func (UpdateKey) ActionFor(urn.Key) {}
func (UpdateKey) String() string    { return "update_key" }

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

// DeleteKey authorizes deleting key resources.
//
// Valid resource: urn.Key.
type DeleteKey struct{}

func (DeleteKey) ActionFor(urn.Key) {}
func (DeleteKey) String() string    { return "delete_key" }
