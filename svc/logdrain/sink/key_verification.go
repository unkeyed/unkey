package sink

// KeyVerificationPayload is the verification event body shared by HTTP and Axiom.
type KeyVerificationPayload struct {
	RequestID    string                  `json:"request_id"`
	KeySpaceID   string                  `json:"key_space_id"`
	Identity     KeyVerificationIdentity `json:"identity"`
	KeyID        string                  `json:"key_id"`
	Region       string                  `json:"region"`
	Source       KeyVerificationSource   `json:"source"`
	Outcome      string                  `json:"outcome"`
	Tags         []string                `json:"tags"`
	SpentCredits int64                   `json:"spent_credits"`
}

func (KeyVerificationPayload) isPayload() {}

type KeyVerificationIdentity struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId"`
}

type KeyVerificationSource struct {
	Type  string `json:"type"`
	AppID string `json:"appId,omitempty"`
}
