package notificationapi_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/redaction"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestDestinationResponsesExposeOnlyTheOrigin(t *testing.T) {
	value, err := notificationapi.Destination(&ctrlv1.NotificationDestination{
		Id:      uid.New(uid.TestPrefix),
		Channel: "webhook",
		UrlHint: "https://user:secret@example.com:443/private?token=secret#secret",
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com:443", value.UrlHint)
	require.NotNil(t, value.HeaderNames)
	_, err = notificationapi.Destination(&ctrlv1.NotificationDestination{Channel: "webhook", UrlHint: "%secret"})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret")
}

func TestNotificationErrorsDoNotRetainTransportSecrets(t *testing.T) {
	for _, code := range []connect.Code{connect.CodeInvalidArgument, connect.CodeNotFound, connect.CodeFailedPrecondition, connect.CodeInternal, connect.CodeUnavailable} {
		err := notificationapi.HandleError(connect.NewError(code, errors.New("https://example.com/SECRET")), "update destination")
		require.NotContains(t, err.Error(), "SECRET")
		require.NotContains(t, fault.UserFacingMessage(err), "SECRET")
		if code == connect.CodeNotFound {
			got, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Data.Notification.NotFound.URN(), got)
			require.Equal(t, "Notification resource not found.", fault.UserFacingMessage(err))
		}
	}
}

func TestUnrelatedPreconditionsDoNotClaimDestinationReferences(t *testing.T) {
	for _, operation := range []string{"list notification members", "create notification rule", "update personal notification preferences"} {
		err := notificationapi.HandleError(connect.NewError(connect.CodeFailedPrecondition, errors.New("internal configuration or limit SECRET")), operation)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
		require.Equal(t, "Failed to "+operation+".", fault.UserFacingMessage(err))
		require.NotContains(t, err.Error(), "SECRET")
	}
}

func TestNotificationWritesAreRedactedFromLogs(t *testing.T) {
	paths, err := redaction.PathsFromSpec(openapi.Spec)
	require.NoError(t, err)
	value := "SECRET"
	request := openapi.V2NotificationsCreateDestinationRequestBody{
		Name:           "delivery",
		Channel:        openapi.Webhook,
		Url:            new("https://example.com/SECRET"),
		SigningSecret:  new(strings.Repeat(value, 8)),
		Headers:        &[]openapi.NotificationHeader{{Name: "Authorization", Value: &value}},
		IdempotencyKey: uid.New(uid.TestPrefix),
	}
	body, err := json.Marshal(request)
	require.NoError(t, err)
	redacted := redaction.New(paths).Redact(body)
	require.NotContains(t, string(redacted), value)
	require.Contains(t, string(redacted), "Authorization")
}

func TestHeaderUpdatesPreserveOmittedVersusEmpty(t *testing.T) {
	require.Nil(t, notificationapi.Headers(nil))
	empty := []openapi.NotificationHeader{}
	require.NotNil(t, notificationapi.Headers(&empty))
	require.Empty(t, notificationapi.Headers(&empty).GetItems())
}

func TestRuleTargetsRoundTripWithoutLosingScope(t *testing.T) {
	value := &ctrlv1.NotificationRule{
		Id:            uid.New(uid.TestPrefix),
		ProjectId:     uid.New(uid.ProjectPrefix),
		AppId:         uid.New(uid.AppPrefix),
		EnvironmentId: uid.New(uid.EnvironmentPrefix),
		Targets: []*ctrlv1.NotificationTarget{
			{Target: &ctrlv1.NotificationTarget_DestinationId{DestinationId: uid.New(uid.TestPrefix)}},
			{Target: &ctrlv1.NotificationTarget_UserId{UserId: uid.New(uid.TestPrefix)}},
			{Target: &ctrlv1.NotificationTarget_Audience{Audience: notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS}},
		},
	}
	public, err := notificationapi.Rule(value)
	require.NoError(t, err)
	require.Equal(t, value.ProjectId, *public.ProjectId)
	require.Equal(t, value.AppId, *public.AppId)
	require.Equal(t, value.EnvironmentId, *public.EnvironmentId)
	targets, err := notificationapi.Targets(public.Targets)
	require.NoError(t, err)
	require.Equal(t, value.Targets, targets)
}
