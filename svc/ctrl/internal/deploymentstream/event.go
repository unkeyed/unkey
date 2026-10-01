package deploymentstream

// Event contains either a deployment ID to look up or a checkpoint token.
type Event struct {
	DeploymentID string
	ResumeToken  []byte
	HasBefore    bool
}
