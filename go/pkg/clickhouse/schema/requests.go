package schema

type ApiRequestV1 struct {
	RequestID       string   `ch:"request_id"`
	Time            int64    `ch:"time"`
	Host            string   `ch:"host"`
	Method          string   `ch:"method"`
	Path            string   `ch:"path"`
	RequestHeaders  []string `ch:"request_headers"`
	RequestBody     string   `ch:"request_body"`
	ResponseStatus  int      `ch:"response_status"`
	ResponseHeaders []string `ch:"response_headers"`
	ResponseBody    string   `ch:"response_body"`
	Error           string   `ch:"error"`
	ServiceLatency  int64    `ch:"serviceLatency"`
}

type KeyVerificationRequestV1 struct {
	RequestID   string `ch:"request_id"`
	Time        int64  `ch:"time"`
	WorkspaceID string `ch:"workspace_id"`
	KeySpaceID  string `ch:"key_space_id"`
	KeyID       string `ch:"key_id"`
	Region      string `ch:"region"`
	Outcome     string `ch:"outcome"`
	IdentityID  string `ch:"identity_id"`
}
