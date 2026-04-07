package domainconnect

import "errors"

var (
	// ErrNoDomainConnectRecord is returned when the domain has no _domainconnect TXT record.
	ErrNoDomainConnectRecord = errors.New("no _domainconnect TXT record found")
	// ErrNoDomainConnectSettings is returned when the Domain Connect settings endpoint is unavailable.
	ErrNoDomainConnectSettings = errors.New("domain connect settings unavailable")
)
