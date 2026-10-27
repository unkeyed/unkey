package notificationapi

import (
	"errors"
	"net/url"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func HandleError(err error, operation string) error {
	if connect.CodeOf(err) == connect.CodeNotFound {
		return fault.New("notification resource not found",
			fault.Code(codes.Data.Notification.NotFound.URN()),
			fault.Public("Notification resource not found."),
		)
	}
	return ctrlclient.HandleError(connect.NewError(connect.CodeOf(err), errors.New("notification service request failed")), operation)
}

func Destination(value *ctrlv1.NotificationDestination) (openapi.NotificationDestination, error) {
	origin, err := url.Parse(value.GetUrlHint())
	if err != nil || origin.Scheme != "https" || origin.Host == "" {
		return openapi.NotificationDestination{}, invalidResponse()
	}
	channel := openapi.NotificationChannel(value.GetChannel())
	if !channel.Valid() {
		return openapi.NotificationDestination{}, invalidResponse()
	}
	return openapi.NotificationDestination{
		Id:          value.GetId(),
		Name:        value.GetName(),
		Channel:     channel,
		Enabled:     value.GetEnabled(),
		UrlHint:     (&url.URL{Scheme: origin.Scheme, Host: origin.Host}).String(),
		HeaderNames: append([]string{}, value.GetHeaderNames()...),
		CreatedAt:   value.GetCreatedAt(),
	}, nil
}

func Headers(values *[]openapi.NotificationHeader) *ctrlv1.NotificationHeaders {
	if values == nil {
		return nil
	}
	headers := make([]*ctrlv1.NotificationHeader, 0, len(*values))
	for _, value := range *values {
		headers = append(headers, &ctrlv1.NotificationHeader{
			Name:  value.Name,
			Value: ptr.SafeDeref(value.Value, ""),
		})
	}
	return &ctrlv1.NotificationHeaders{Items: headers}
}

func Targets(values []openapi.NotificationTarget) ([]*ctrlv1.NotificationTarget, error) {
	targets := make([]*ctrlv1.NotificationTarget, 0, len(values))
	for _, value := range values {
		variant, err := value.ValueByDiscriminator()
		if err != nil {
			return nil, invalidTarget()
		}
		switch target := variant.(type) {
		case openapi.NotificationDestinationTarget:
			targets = append(targets, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_DestinationId{DestinationId: target.DestinationId}})
		case openapi.NotificationUserTarget:
			targets = append(targets, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_UserId{UserId: target.UserId}})
		case openapi.NotificationAudienceTarget:
			if target.Audience != openapi.NotificationAudienceTargetAudienceWorkspaceAdmins {
				return nil, invalidTarget()
			}
			targets = append(targets, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_Audience{Audience: notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS}})
		default:
			return nil, invalidTarget()
		}
	}
	return targets, nil
}

func Rule(value *ctrlv1.NotificationRule) (openapi.NotificationRule, error) {
	targets := make([]openapi.NotificationTarget, 0, len(value.GetTargets()))
	for _, target := range value.GetTargets() {
		var result openapi.NotificationTarget
		var err error
		switch variant := target.GetTarget().(type) {
		case *ctrlv1.NotificationTarget_DestinationId:
			err = result.FromNotificationDestinationTarget(openapi.NotificationDestinationTarget{
				Type:          openapi.Destination,
				DestinationId: variant.DestinationId,
			})
		case *ctrlv1.NotificationTarget_UserId:
			err = result.FromNotificationUserTarget(openapi.NotificationUserTarget{
				Type:   openapi.NotificationUserTargetTypeUser,
				UserId: variant.UserId,
			})
		case *ctrlv1.NotificationTarget_Audience:
			if variant.Audience != notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS {
				return openapi.NotificationRule{}, invalidResponse()
			}
			err = result.FromNotificationAudienceTarget(openapi.NotificationAudienceTarget{
				Type:     openapi.Audience,
				Audience: openapi.NotificationAudienceTargetAudienceWorkspaceAdmins,
			})
		default:
			return openapi.NotificationRule{}, invalidResponse()
		}
		if err != nil {
			return openapi.NotificationRule{}, invalidResponse()
		}
		targets = append(targets, result)
	}
	return openapi.NotificationRule{
		Id:               value.GetId(),
		Name:             value.GetName(),
		NotificationType: value.GetNotificationType(),
		ProjectId:        OptionalString(value.GetProjectId()),
		AppId:            OptionalString(value.GetAppId()),
		EnvironmentId:    OptionalString(value.GetEnvironmentId()),
		Enabled:          value.GetEnabled(),
		Targets:          targets,
	}, nil
}

func Preferences(values []*ctrlv1.NotificationPreference) []openapi.NotificationPreference {
	preferences := make([]openapi.NotificationPreference, 0, len(values))
	for _, value := range values {
		preferences = append(preferences, openapi.NotificationPreference{
			NotificationType: value.GetNotificationType(),
			Enabled:          value.GetEnabled(),
		})
	}
	return preferences
}

func OptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func invalidResponse() error {
	return fault.New("invalid notification service response",
		fault.Code(codes.App.Internal.UnexpectedError.URN()),
		fault.Public("Unable to read the notification service response."),
	)
}

func invalidTarget() error {
	return fault.New("invalid notification target",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public("Provide a destination, user, or workspace_admins audience target."),
	)
}
