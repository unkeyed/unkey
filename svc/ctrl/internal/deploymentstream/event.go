package deploymentstream

// Event contains either a deployment ID to look up or a checkpoint token.
// Watch sets exactly one of these fields.
type Event struct {
	DeploymentID string
	ResumeToken  []byte
}
