package deployment

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
)

// RejectionFault turns a refused create into the error its caller sees.
//
// The worker answers a refusal with an enum and keeps the explanation in its
// own logs, because that text names repositories and deployments the caller may
// have no right to read. So the wording here is written from the reason alone
// and stays deliberately coarser than a handler's own pre-flight, which can
// name the offending field because it read it.
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
			fault.Public("This app has no connected GitHub repository. Deploy a prebuilt image with the image source, or connect a repository first."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_COMMIT_NOT_RESOLVED:
		return fault.New(
			"commit not resolved",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: github could not resolve the branch or commit"),
			fault.Public("The requested branch or commit could not be resolved on GitHub."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_SOURCE_IMAGE:
		return fault.New(
			"no source image",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: nothing to build from"),
			fault.Public("This deployment cannot be redeployed because it never produced an image."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NEWER_DEPLOYMENT_EXISTS:
		return fault.New(
			"newer deployment exists",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: a newer active deployment exists"),
			fault.Public("A newer deployment already exists for this app, environment, and branch."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_ENVIRONMENT_NOT_DEPLOYABLE:
		return fault.New(
			"environment not deployable",
			fault.Code(codes.App.Validation.InvalidEnvironmentSettings.URN()),
			fault.Internal("create rejected: environment runtime or regional settings are out of bounds"),
			fault.Public("This environment cannot be deployed. Update its runtime and region settings before deploying."),
		)

	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_INVALID_IMAGE:
		return fault.New(
			"invalid image",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("create rejected: image is not a well-formed container reference"),
			fault.Public("The image is not a valid container reference."),
		)

	// Both mean the target vanished between this handler's own lookup and the
	// create. They answer alike so neither confirms the existence of something
	// the caller cannot reach.
	case hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_TARGET_NOT_FOUND,
		hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SOURCE_DEPLOYMENT_NOT_FOUND:
		return fault.New(
			"deployment target not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("create rejected: target or source deployment does not exist"),
			fault.Public("The requested project, app, environment, or deployment does not exist."),
		)

	// A refusal the worker could name but this mapping cannot is a bug here, not
	// a caller error: the deployment was refused and the caller deserves to know
	// it failed rather than read a 201 for a row that will never exist.
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
