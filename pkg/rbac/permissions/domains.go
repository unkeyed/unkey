package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateDomain authorizes attaching a custom domain to an environment.
//
// Valid resource: urn.Domain. Grants use a wildcard domain id because the domain
// does not exist when the request is authorized.
type CreateDomain struct{}

func (CreateDomain) ActionFor(urn.Domain) {}
func (CreateDomain) String() string       { return "create_domain" }

// ReadDomain authorizes reading a custom domain.
//
// Valid resource: urn.Domain.
type ReadDomain struct{}

func (ReadDomain) ActionFor(urn.Domain) {}
func (ReadDomain) String() string       { return "read_domain" }

// WriteDomain authorizes creating or updating a domain.
type WriteDomain struct{}

func (WriteDomain) ActionFor(urn.Domain) {}
func (WriteDomain) String() string       { return "write_domain" }

// DeleteDomain authorizes removing a custom domain.
//
// Valid resource: urn.Domain.
type DeleteDomain struct{}

func (DeleteDomain) ActionFor(urn.Domain) {}
func (DeleteDomain) String() string       { return "delete_domain" }

// VerifyDomain authorizes restarting verification for a custom domain.
//
// Valid resource: urn.Domain.
type VerifyDomain struct{}

func (VerifyDomain) ActionFor(urn.Domain) {}
func (VerifyDomain) String() string       { return "verify_domain" }
