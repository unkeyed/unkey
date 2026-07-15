package email

// resendTemplate is the nested template object in the send payload. Resend
// rejects a payload that mixes a template with html/text, so this is the only
// content carrier.
type resendTemplate struct {
	ID        string            `json:"id"`
	Variables map[string]string `json:"variables,omitempty"`
}

type resendRequest struct {
	// From and Subject are omitempty so a template-only send can leave them out
	// and let the published template's own From and Subject apply. Setting either
	// overrides the template.
	From     string         `json:"from,omitempty"`
	To       []string       `json:"to"`
	Subject  string         `json:"subject,omitempty"`
	Template resendTemplate `json:"template"`
}

func newResendRequest(from string, email Email) resendRequest {
	return resendRequest{
		From:     from,
		To:       email.To,
		Subject:  email.Subject,
		Template: resendTemplate{ID: email.TemplateID, Variables: email.Variables},
	}
}
