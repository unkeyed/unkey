package urn

import "fmt"

// billing builds billing resource paths.
//
// Hierarchy:
//
//	workspace
//	└── billing
type billing struct {
	workspaceID string
	path        string
}

// Invoice returns a billing invoice resource path.
//
// Subresource:
//
//	billing
//	└── invoices/{invoice_id}
func (b billing) Invoice(invoiceID string) V1 {
	return V1{
		WorkspaceID: b.workspaceID,
		Resource:    fmt.Sprintf("%s/invoices/%s", b.path, invoiceID),
	}
}

// Quotas returns the workspace billing quotas resource path.
//
// Subresource:
//
//	billing
//	└── quotas
func (b billing) Quotas() V1 {
	return V1{
		WorkspaceID: b.workspaceID,
		Resource:    b.path + "/quotas",
	}
}
