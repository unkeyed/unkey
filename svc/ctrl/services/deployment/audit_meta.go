package deployment

// deploymentAuditMeta is the resource metadata every deployment audit entry
// this service writes must carry. The dashboard's audit feed filters on these
// keys, so authorize and cancel produce the same shape rather than each
// spelling out its own map.
func deploymentAuditMeta(projectID, appID, environmentID string) map[string]any {
	return map[string]any{
		"projectId":     projectID,
		"appId":         appID,
		"environmentId": environmentID,
	}
}
