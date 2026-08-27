package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadDomain authorizes reading a domain.
type ReadDomain struct{}

func (ReadDomain) ActionFor(urn.Domain) {}
func (ReadDomain) String() string       { return "read_domain" }

// WriteDomain authorizes creating or updating a domain.
type WriteDomain struct{}

func (WriteDomain) ActionFor(urn.Domain) {}
func (WriteDomain) String() string       { return "write_domain" }

// DeleteDomain authorizes deleting a domain.
type DeleteDomain struct{}

func (DeleteDomain) ActionFor(urn.Domain) {}
func (DeleteDomain) String() string       { return "delete_domain" }
