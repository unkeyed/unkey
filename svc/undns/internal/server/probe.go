package server

import (
	"context"
	"fmt"
	"time"

	dnswire "codeberg.org/miekg/dns"
)

// Probe checks that a DNS server answers on network ("udp" or "tcp") at
// address within 500 milliseconds. It sends a query without a question, which
// a healthy server rejects with FORMERR without consulting discovery or the
// upstream resolver.
func Probe(ctx context.Context, network, address string) error {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	query := new(dnswire.Msg)
	query.ID = dnswire.ID()
	query.RecursionDesired = true
	client := dnswire.NewClient()
	client.ReadTimeout = 500 * time.Millisecond
	client.WriteTimeout = 500 * time.Millisecond
	response, _, err := client.Exchange(ctx, query, network, address)
	if err != nil {
		return fmt.Errorf("probe DNS service: %w", err)
	}
	if response.Rcode != dnswire.RcodeFormatError {
		return fmt.Errorf("probe DNS service returned %s", dnswire.RcodeToString[response.Rcode])
	}
	return nil
}
