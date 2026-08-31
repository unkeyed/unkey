package permissions

import "github.com/unkeyed/unkey/pkg/urn"

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
