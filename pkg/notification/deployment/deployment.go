package deployment

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/notification"
)

const ServiceName = "notification.v1.DeploymentService"

type Change string

const (
	Preparing                 Change = "preparing"
	BuildFailed               Change = "build_failed"
	LoadFailed                Change = "load_failed"
	Cancelled                 Change = "cancelled"
	Superseded                Change = "superseded"
	RegionsStarted            Change = "regions_started"
	RegionsFailed             Change = "regions_failed"
	RoutingStarted            Change = "routing_started"
	RoutingFailed             Change = "routing_failed"
	FinalizationFailed        Change = "finalization_failed"
	Live                      Change = "live"
	CompletedWithoutPromotion Change = "completed_without_promotion"
)

type ReportRequest struct {
	DeploymentID string `json:"deploymentId"`
	Change       Change `json:"change"`
}

type Payload struct {
	DeploymentID   string `json:"deploymentId"`
	Change         Change `json:"change"`
	Project        string `json:"project"`
	App            string `json:"app"`
	Environment    string `json:"environment"`
	EnvironmentURL string `json:"environmentUrl"`
	LogURL         string `json:"logUrl"`
}

// Report journals a background report; GitHub or subscription delivery failures
// do not fail the deployment. Call it outside restate.Run.
func Report(ctx restate.Context, deploymentID string, change Change) {
	restate.Object[string](ctx, ServiceName, deploymentID, "Report").Send(
		ReportRequest{DeploymentID: deploymentID, Change: change},
		restate.WithIdempotencyKey(string(change)),
	)
}

func (c Change) Revision() int {
	switch c {
	case Preparing:
		return 1
	case BuildFailed:
		return 2
	case LoadFailed:
		return 3
	case Cancelled, Superseded:
		return 4
	case RegionsStarted:
		return 5
	case RegionsFailed:
		return 6
	case RoutingStarted:
		return 7
	case RoutingFailed:
		return 8
	case FinalizationFailed:
		return 9
	case Live, CompletedWithoutPromotion:
		return 10
	}
	return 0
}

func (c Change) Terminal() bool {
	switch c {
	case BuildFailed, LoadFailed, Cancelled, Superseded, RegionsFailed, RoutingFailed, FinalizationFailed, Live, CompletedWithoutPromotion:
		return true
	case Preparing, RegionsStarted, RoutingStarted:
		return false
	}
	return false
}

func (c Change) Type() string {
	switch c {
	case Preparing:
		return "deployment.preparing"
	case RegionsStarted:
		return "deployment.started"
	case RoutingStarted:
		return "deployment.routing"
	case BuildFailed, LoadFailed, RegionsFailed, RoutingFailed, FinalizationFailed:
		return "deployment.failed"
	case Cancelled:
		return "deployment.cancelled"
	case Superseded:
		return "deployment.superseded"
	case Live, CompletedWithoutPromotion:
		return "deployment.succeeded"
	}
	return ""
}

func (c Change) Description() string {
	switch c {
	case Preparing:
		return "Preparing deployment"
	case BuildFailed:
		return "Build or image preparation failed"
	case LoadFailed:
		return "Could not load deployment after build"
	case Cancelled:
		return "Deployment stopped: cancelled"
	case Superseded:
		return "Deployment stopped: superseded"
	case RegionsStarted:
		return "Deploying to regions"
	case RegionsFailed:
		return "Deployment to regions failed"
	case RoutingStarted:
		return "Configuring routing"
	case RoutingFailed:
		return "Routing configuration failed"
	case FinalizationFailed:
		return "Finalization failed"
	case Live:
		return "Deployment is live"
	case CompletedWithoutPromotion:
		return "Deployment completed without changing live traffic"
	}
	return ""
}

func Definitions() []notification.Definition {
	return []notification.Definition{
		definition{eventType: Preparing.Type()},
		definition{eventType: RegionsStarted.Type()},
		definition{eventType: RoutingStarted.Type()},
		definition{eventType: RegionsFailed.Type()},
		definition{eventType: Cancelled.Type()},
		definition{eventType: Superseded.Type()},
		definition{eventType: Live.Type()},
	}
}

func Decode(event notification.Event) (Payload, error) {
	decoder := notification.JSONDefinition[Payload]{
		Name: event.Type, SchemaVersion: 1, RenderPayload: nil,
	}
	payload, err := decoder.Decode(event)
	if err != nil {
		return payload, err
	}
	if payload.Change.Revision() == 0 || payload.Change.Type() != event.Type || payload.DeploymentID == "" ||
		payload.Project == "" || payload.App == "" || payload.Environment == "" ||
		event.SourceEventID != payload.DeploymentID+":"+string(payload.Change) ||
		event.Resource.ProjectID == "" || event.Resource.AppID == "" || event.Resource.EnvironmentID == "" {
		return payload, errors.New("invalid deployment notification")
	}
	for _, raw := range []string{payload.EnvironmentURL, payload.LogURL} {
		if raw == "" {
			continue
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return payload, errors.New("invalid deployment notification URL")
		}
	}
	return payload, nil
}

func Event(workspaceID string, resource notification.Resource, occurredAtMillis int64, payload Payload) (notification.Event, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return notification.Event{}, err
	}
	event := notification.Event{
		WorkspaceID: workspaceID, SourceEventID: payload.DeploymentID + ":" + string(payload.Change),
		Type: payload.Change.Type(), SchemaVersion: 1, Resource: resource, Payload: encoded,
		OccurredAt: occurredAtMillis, ExpiresAt: 0,
	}
	if _, err := Decode(event); err != nil {
		return notification.Event{}, err
	}
	return event.Normalize()
}

type definition struct {
	eventType string
}

func (d definition) Type() string { return d.eventType }

func (d definition) Render(event notification.Event) (notification.Content, error) {
	if event.Type != d.eventType {
		return notification.Content{}, errors.New("unexpected deployment notification type")
	}
	payload, err := Decode(event)
	if err != nil {
		return notification.Content{}, err
	}
	text := fmt.Sprintf("%s / %s (%s): %s.", payload.Project, payload.App, payload.Environment, payload.Change.Description())
	if payload.EnvironmentURL != "" {
		text += "\nVisit: " + payload.EnvironmentURL
	}
	if payload.LogURL != "" {
		text += "\nInspect: " + payload.LogURL
	}
	return notification.Content{Subject: payload.Change.Description(), Text: text, HTML: ""}, nil
}
