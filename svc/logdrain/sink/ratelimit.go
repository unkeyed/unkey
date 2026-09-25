package sink

// RatelimitPayload exports one API rate-limit decision, not configuration audit events.
type RatelimitPayload struct {
	RequestID   string `json:"request_id"`
	NamespaceID string `json:"namespace_id"`
	Identifier  string `json:"identifier"`
	Passed      bool   `json:"passed"`
	OverrideID  string `json:"override_id,omitempty"`
	Limit       uint64 `json:"limit"`
	Remaining   uint64 `json:"remaining"`
	ResetAt     int64  `json:"reset_at"`
	Tokens      uint64 `json:"tokens"`
	Source      string `json:"source"`
}

func (RatelimitPayload) isPayload() {}
