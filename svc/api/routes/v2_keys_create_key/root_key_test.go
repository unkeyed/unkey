package handler_test

import "fmt"

func rootKeyGrant(workspaceID, projectID, keyspaceID, action string) string {
	return fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/*#%s", workspaceID, projectID, keyspaceID, action)
}
