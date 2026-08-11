package project

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

func TestCreateProjectRejectsReservedSlug(t *testing.T) {
	const bearer = "test-token"
	svc := New(Config{Bearer: bearer})

	for _, slug := range []string{"default", "Default"} {
		t.Run(slug, func(t *testing.T) {
			req := connect.NewRequest(&ctrlv1.CreateProjectRequest{
				WorkspaceId: "ws_test",
				Name:        "Reserved",
				Slug:        slug,
				Actor:       &ctrlv1.ActorInfo{},
			})
			req.Header().Set("Authorization", "Bearer "+bearer)

			_, err := svc.CreateProject(t.Context(), req)
			require.Error(t, err)
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			require.ErrorContains(t, err, "slug is reserved")
		})
	}
}
