// Package email sends transactional email through a provider. Message content
// lives in published provider templates, not in this repo: a caller names a
// template and supplies the variables it expects, so changing copy is a
// template edit, not a code change.
//
// The package is transport only. It does not own templates, recipient lists,
// or the decision of who to email; callers resolve those and hand a fully
// addressed [Email] to a [Sender]. A [NewNoop] sender logs instead of sending,
// for local and any environment without a provider key, mirroring the
// noop-or-real split used elsewhere in the control plane (billingmeter,
// invoicecloser).
package email

import "context"

// Email is one transactional message rendered from a published template. The
// content is the template's (referenced by TemplateID); the caller supplies
// the recipients and the variables the template declares.
type Email struct {
	// From is the sender address, e.g. "Unkey <billing@unkey.com>". Empty uses
	// the Sender's configured default.
	From string
	// To is the recipient list; at least one address is required.
	To []string
	// TemplateID is the published template to render. A draft template cannot
	// be sent and the provider rejects it.
	TemplateID string
	// Variables fills the template's declared variables. Keys must match the
	// template exactly; the provider rejects a send with missing variables.
	Variables map[string]string
	// Subject overrides the template's default subject when set.
	Subject string
}

// Sender delivers an [Email] through a provider. Implementations are
// provider-specific; see [NewResend] for the real one and [NewNoop] for the
// logging stub.
type Sender interface {
	Send(ctx context.Context, email Email) error
}
