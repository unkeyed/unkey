package deployment

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/unkeyed/unkey/pkg/notification"
)

// Payload is the version-one deployment notification body.
// EnvironmentURL and LogURL are optional HTTP or HTTPS links without user credentials.
type Payload struct {
	DeploymentID   string `json:"deploymentId"`
	Change         Change `json:"change"`
	Project        string `json:"project"`
	App            string `json:"app"`
	Environment    string `json:"environment"`
	EnvironmentURL string `json:"environmentUrl"`
	LogURL         string `json:"logUrl"`
}

// Definitions returns the renderers for all supported deployment event types.
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

// Decode validates a version-one payload, its source identity, and resource scope.
// It may return a partial payload on error; it does not validate the whole envelope.
func Decode(event notification.Event) (Payload, error) {
	decoder := notification.JSONDefinition[Payload]{
		Name:          event.Type,
		SchemaVersion: 1,
		RenderPayload: nil,
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

// Event builds and validates a version-one event with no expiry.
// Each deployment/change pair has a distinct source identity; errors return a zero event.
func Event(workspaceID string, resource notification.Resource, occurredAtMillis int64, payload Payload) (notification.Event, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return notification.Event{}, err
	}
	event := notification.Event{
		WorkspaceID:   workspaceID,
		SourceEventID: payload.DeploymentID + ":" + string(payload.Change),
		Type:          payload.Change.Type(),
		SchemaVersion: 1,
		Resource:      resource,
		Payload:       encoded,
		OccurredAt:    occurredAtMillis,
		ExpiresAt:     0,
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
