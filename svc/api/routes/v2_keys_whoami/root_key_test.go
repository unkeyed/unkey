package handler_test

import "fmt"

func rootKeyGrant(workspaceID, projectID, keyspaceID, keyID, action string) string {
	return fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#%s", workspaceID, projectID, keyspaceID, keyID, action)
}
