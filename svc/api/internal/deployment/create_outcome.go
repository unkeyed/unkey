package deployment

import (
	"fmt"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
)

// errorForOutcome maps a create outcome onto the error the caller sees, nil for
// a created row. The worker's detail can name repositories and deployments the
// caller may not read, so it is shown only for outcomes about the caller's own
// input or settings; every other message is written from the outcome alone.
func errorForOutcome(outcome hydrav1.CreateOutcome, detail string) error {
	switch outcome {
	case hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED:
		return nil

	case hydrav1.CreateOutcome_CREATE_OUTCOME_NO_COMPUTE_PLAN:
		return fault.New(
			"workspace has no Compute plan",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: workspace has no Compute plan"),
			fault.Public(deploygate.MsgNoComputePlan),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_SPEND_SUSPENDED:
		return fault.New(
			"workspace is spend suspended",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: workspace is over its Compute spend cap"),
			fault.Public(deploygate.StartSpendSuspended.Message()),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_NO_REPO_CONNECTION:
		return fault.New(
			"no repo connection",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: app has no github repo connection"),
			fault.Public("This app has no GitHub repository connected. Connect one, or deploy a prebuilt image instead."),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_COMMIT_NOT_RESOLVED:
		return fault.New(
			"commit not resolved",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: github could not resolve the branch or commit: "+detail),
			fault.Public(fmt.Sprintf("GitHub could not find the branch or commit: %s. Check the name, and that Unkey still has access to the repository.", detail)),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_NO_SOURCE_IMAGE:
		return fault.New(
			"no source image",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: nothing to build from"),
			fault.Public("That deployment never finished building, so there is nothing to redeploy. Choose a deployment that succeeded, or deploy an image."),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_NO_SOURCE_COMMIT:
		return fault.New(
			"no source commit",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: git deployment recorded no commit"),
			fault.Public("That deployment was a Git build but recorded no commit, so there is nothing to rebuild. Choose another deployment, or deploy a branch or commit."),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_NO_IMAGE_CONFIGURED:
		return fault.New(
			"no image configured",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: OCI app has no image configured"),
			fault.Public("This app deploys a container image but none is configured. Set an image on the app, or pass one in the request."),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_NO_SOURCE:
		return fault.New(
			"no source",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: no source in the request and nothing to fall back to"),
			fault.Public("Nothing to deploy. Pass a git, image, or deployment source; this app has no repository connected and has never deployed."),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_NEWER_DEPLOYMENT_EXISTS:
		return fault.New(
			"newer deployment exists",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("create rejected: "+detail),
			fault.Public(fmt.Sprintf("A newer deployment has already shipped: %s.", detail)),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_ENVIRONMENT_NOT_DEPLOYABLE:
		return fault.New(
			"environment not deployable",
			fault.Code(codes.App.Validation.InvalidEnvironmentSettings.URN()),
			fault.Internal("create rejected: environment runtime or regional settings are out of bounds: "+detail),
			fault.Public(fmt.Sprintf("This environment cannot be deployed: %s. Update the environment's settings before deploying.", detail)),
		)

	case hydrav1.CreateOutcome_CREATE_OUTCOME_INVALID_IMAGE:
		return fault.New(
			"invalid image",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("create rejected: image is not a well-formed container reference: "+detail),
			fault.Public(fmt.Sprintf("The docker image is not valid: %s. Expected [registry/]repository[:tag][@digest], for example ghcr.io/acme/api:v1.2.3.", detail)),
		)

	// One answer for both, so neither confirms that something the caller cannot
	// reach exists.
	case hydrav1.CreateOutcome_CREATE_OUTCOME_TARGET_NOT_FOUND,
		hydrav1.CreateOutcome_CREATE_OUTCOME_SOURCE_DEPLOYMENT_NOT_FOUND:
		return fault.New(
			"deployment target not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("create rejected: target or source deployment does not exist"),
			fault.Public("The project, app, environment, or deployment does not exist."),
		)

	// A rejection this mapping cannot name is still a rejection: answer an
	// error, never a 201 for a row that does not exist.
	case hydrav1.CreateOutcome_CREATE_OUTCOME_UNSPECIFIED:
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
			fault.Internal("create rejected with an outcome svc/api does not map: "+outcome.String()),
			fault.Public("Failed to create deployment."),
		)
	}
}
