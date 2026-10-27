package githubstatus

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/notification"
)

const commentServiceName = "notification.v1.GitHubPullRequestService"

type commentRequest struct {
	Target reportContext      `json:"target"`
	Event  notification.Event `json:"event"`
}

// CommentDefinition binds the private object shared by all deployments for a PR.
func (s *Service) CommentDefinition() restate.ServiceDefinition {
	return restate.NewObject(commentServiceName).
		Handler("Update", restate.NewObjectHandler(s.UpdateComment, restate.WithIngressPrivate(true), restate.WithInvocationRetryPolicy(
			restate.WithInitialRetryInterval(time.Second), restate.WithMaxRetryInterval(time.Minute),
			restate.WithMaxRetryAttempts(20), restate.PauseOnMaxAttempts(),
		)))
}

func commentKey(target reportContext) string {
	return fmt.Sprintf("%d:%d:%d", target.InstallationID, target.RepositoryID, target.PullRequest)
}
