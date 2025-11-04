package openapi

import (
	"context"
	"encoding/base64"

	"connectrpc.com/connect"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oasdiff/oasdiff/checker"
	"github.com/oasdiff/oasdiff/diff"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (s *Service) GetOpenApiDiff(ctx context.Context, req *connect.Request[ctrlv1.GetOpenApiDiffRequest]) (*connect.Response[ctrlv1.GetOpenApiDiffResponse], error) {
	// Load old version spec
	oldSpec, err := s.loadOpenApiSpec(ctx, req.Msg.GetOldDeploymentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fault.Wrap(err,
			fault.Internal("failed to load old version spec"),
			fault.Public("Old version not found"),
		))
	}

	// Load new version spec
	newSpec, err := s.loadOpenApiSpec(ctx, req.Msg.GetNewDeploymentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fault.Wrap(err,
			fault.Internal("failed to load new version spec"),
			fault.Public("New version not found"),
		))
	}

	// Parse OpenAPI specs
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	b1, err := base64.StdEncoding.DecodeString(oldSpec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fault.Wrap(err,
			fault.Internal("failed to decode old version spec"),
			fault.Public("Invalid base64 encoding in old version"),
		))
	}

	s1, err := loader.LoadFromData(b1)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fault.Wrap(err,
			fault.Internal("failed to parse old OpenAPI spec"),
			fault.Public("Invalid OpenAPI specification in old version"),
		))
	}

	b2, err := base64.StdEncoding.DecodeString(newSpec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fault.Wrap(err,
			fault.Internal("failed to decode new version spec"),
			fault.Public("Invalid base64 encoding in new version"),
		))
	}

	s2, err := loader.LoadFromData(b2)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fault.Wrap(err,
			fault.Internal("failed to parse new OpenAPI spec"),
			fault.Public("Invalid OpenAPI specification in new version"),
		))
	}

	// Generate diff report
	// nolint: exhaustruct
	diffReport, err := diff.Get(&diff.Config{}, s1, s2)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fault.Wrap(err,
			fault.Internal("failed to generate diff report"),
			fault.Public("Failed to generate diff report"),
		))
	}

	// Generate changelog using checker
	config := checker.NewConfig(checker.GetAllChecks())
	changes := checker.CheckBackwardCompatibility(
		config,
		diffReport,
		&diff.OperationsSourcesMap{},
	)

	// Check if there are any breaking changes
	hasBreakingChanges := false
	for _, change := range changes {
		if change.GetLevel() == checker.ERR {
			hasBreakingChanges = true
			break
		}
	}

	return connect.NewResponse(&ctrlv1.GetOpenApiDiffResponse{
		Summary:            convertSummaryToProto(diffReport.GetSummary()),
		HasBreakingChanges: hasBreakingChanges,
		Changes:            convertChangesToProto(changes),
	}), nil
}
