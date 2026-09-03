package deployment

// deploymentAuditMeta builds the resource metadata for a deployment audit
// entry. The dashboard's audit feed filters on these keys, so every entry this
// service writes carries all three.
func deploymentAuditMeta(projectID, appID, environmentID string) map[string]any {
	return map[string]any{
		"projectId":     projectID,
		"appId":         appID,
		"environmentId": environmentID,
	}
}
