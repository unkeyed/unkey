package deployment

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
)

// RejectionFault maps a refused create onto the error the caller sees. Only the
// enum crosses the wire, because the worker's detail can name repositories and
// deployments the caller may not read, so each message is written from the
// reason alone.
func RejectionFault(reason hydrav1.CreateRejectionReason) error {
	switch reason {
	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_COMPUTE_PLAN:
		return fault.New(
			"workspace has no Compute plan",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: workspace has no Compute plan"),
			fault.Public(deploygate.MsgNoComputePlan),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SPEND_SUSPENDED:
		return fault.New(
			"workspace is spend suspended",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: workspace is over its Compute spend cap"),
			fault.Public(deploygate.StartSpendSuspended.Message()),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_REPO_CONNECTION:
		return fault.New(
			"no repo connection",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: app has no github repo connection"),
			fault.Public("This app has no GitHub repository connected. Connect one, or deploy a prebuilt image instead."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_COMMIT_NOT_RESOLVED:
		return fault.New(
			"commit not resolved",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: github could not resolve the branch or commit"),
			fault.Public("GitHub could not find that branch or commit. Check the name, and that Unkey still has access to the repository."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_SOURCE_IMAGE:
		return fault.New(
			"no source image",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: nothing to build from"),
			fault.Public("That deployment never finished building, so there is nothing to redeploy. Choose a deployment that succeeded, or deploy an image."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NEWER_DEPLOYMENT_EXISTS:
		return fault.New(
			"newer deployment exists",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: a newer active deployment exists"),
			fault.Public("A newer deployment has already shipped for this app, environment, and branch."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_ENVIRONMENT_NOT_DEPLOYABLE:
		return fault.New(
			"environment not deployable",
			fault.Code(codes.App.Validation.InvalidEnvironmentSettings.URN()),
			fault.Internal("create rejected: environment runtime or regional settings are out of bounds"),
			fault.Public("This environment cannot be deployed yet. Check its port, CPU, memory, and region settings."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_INVALID_IMAGE:
		return fault.New(
			"invalid image",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("create rejected: image is not a well-formed container reference"),
			fault.Public("The docker image is not valid. Expected [registry/]repository[:tag][@digest], for example ghcr.io/acme/api:v1.2.3."),
		)

	// One answer for both, so neither confirms that something the caller cannot
	// reach exists.
	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_TARGET_NOT_FOUND,
		hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SOURCE_DEPLOYMENT_NOT_FOUND:
		return fault.New(
			"deployment target not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("create rejected: target or source deployment does not exist"),
			fault.Public("The project, app, environment, or deployment does not exist."),
		)

	// A rejection this mapping cannot name is still a rejection: answer an
	// error, never a 201 for a row that does not exist.
	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_UNSPECIFIED:
		return fault.New(
			"create rejected without a reason",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("create rejected with an unspecified reason"),
			fault.Public("Failed to create deployment."),
		)

	default:
		return fault.New(
			"unknown create rejection",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("create rejected with a reason svc/api does not map: "+reason.String()),
			fault.Public("Failed to create deployment."),
		)
	}
}
