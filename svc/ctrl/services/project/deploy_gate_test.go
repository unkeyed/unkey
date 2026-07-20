package project

import (
	"context"
	"database/sql"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

func TestDeployEntitled(t *testing.T) {
	null := sql.NullString{Valid: false}
	empty := sql.NullString{Valid: true, String: ""}
	plan := func(s string) sql.NullString { return sql.NullString{Valid: true, String: s} }

	cases := []struct {
		name     string
		plan     sql.NullString
		override sql.NullString
		want     bool
	}{
		{name: "no plan, no override", plan: null, override: null, want: false},
		{name: "empty plan, no override", plan: empty, override: null, want: false},
		{name: "synced plan grants", plan: plan("pro"), override: null, want: true},
		{name: "override grants without plan", plan: null, override: plan("business"), want: true},
		{name: "empty override does not grant", plan: null, override: empty, want: false},
		{name: "both set grants", plan: plan("starter"), override: plan("business"), want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, deployEntitled(tc.plan, tc.override))
		})
	}
}

func TestCreateProjectRejectsReservedSlug(t *testing.T) {
	const bearer = "test-token"
	svc := New(Config{Bearer: bearer}) //nolint:exhaustruct

	for _, slug := range []string{"default", "Default", "DEFAULT"} {
		t.Run(slug, func(t *testing.T) {
			req := connect.NewRequest(&ctrlv1.CreateProjectRequest{
				WorkspaceId: "ws_test",
				Name:        "Reserved",
				Slug:        slug,
				Actor:       &ctrlv1.ActorInfo{}, //nolint:exhaustruct
			})
			req.Header().Set("Authorization", "Bearer "+bearer)

			_, err := svc.CreateProject(context.Background(), req)
			require.Error(t, err)
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}
