package githubstatus

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
)

func gitHubState(change deployment.Change) string {
	switch change {
	case deployment.Preparing:
		return "pending"
	case deployment.Live:
		return "success"
	case deployment.CompletedWithoutPromotion:
		return "inactive"
	case deployment.BuildFailed, deployment.RegionsFailed, deployment.RoutingFailed, deployment.FinalizationFailed:
		return "failure"
	case deployment.LoadFailed, deployment.Cancelled, deployment.Superseded:
		return "error"
	case deployment.RegionsStarted, deployment.RoutingStarted:
		return "in_progress"
	}
	return "error"
}

func gitHubCommitState(state string) string {
	switch state {
	case "success", "inactive":
		return "success"
	case "failure", "error":
		return state
	default:
		return "pending"
	}
}

func environmentLabel(payload deployment.Payload) string {
	label := payload.Project
	if payload.App != "default" {
		label += "/" + payload.App
	}
	return label + " - " + payload.Environment
}

func reportRow(event notification.Event, payload deployment.Payload) string {
	name := payload.Project
	if payload.App != "default" {
		name += " / " + payload.App
	}
	preview := "—"
	if payload.EnvironmentURL != "" {
		preview = fmt.Sprintf("[Visit Preview](%s)", payload.EnvironmentURL)
	}
	return fmt.Sprintf("| %s **%s** (%s) | %s | %s | [Inspect](%s) | %s |",
		rowMarker(event.Resource.AppID, event.Resource.EnvironmentID), name, payload.Environment,
		stateLabel(gitHubState(payload.Change)), preview, payload.LogURL,
		time.UnixMilli(event.OccurredAt).UTC().Format("Jan 2, 2006 3:04pm"))
}
