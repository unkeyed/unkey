import type { Deployment } from "@/lib/collections/deploy/deployments";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { match } from "@unkey/match";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
} from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { type ReactNode, useState } from "react";
import { RedeployDialog } from "../../components/table/components/actions/redeploy-dialog";
import type { StepsData } from "./deployment-progress";
import { LIMITS_DOCS_URL, limitFailure } from "./limit-failure";

/** Patterns matched against backend fault.Public messages to decide whether
 *  to show a "Go to Settings" link. Only errors fixable via project settings
 *  (dockerfile path, docker context, regions, git branch) belong here. */
const SETTINGS_HINT_PATTERNS = [
  // Dockerfile path / docker context (from build.go extractUserBuildError)
  "check that the file path is correct",
  "dockerfile appears to be empty",
  "build target stage was not found",
  "check the root directory",
  // Region configuration (from deploy_handler.go createTopologies)
  "configure at least one region",
  // Git branch (from deploy_handler.go buildImage)
  "git branch could not be resolved",
];

function isSettingsRelatedError(error: string): boolean {
  const lower = error.toLowerCase();
  return SETTINGS_HINT_PATTERNS.some((pattern) => lower.includes(pattern));
}

type StepKey = keyof NonNullable<StepsData>;

// A Record so a step added to the payload fails to compile here rather than
// becoming an unsearched error slot.
const STEP_ORDER: Record<StepKey, number> = {
  queued: 0,
  starting: 1,
  building: 2,
  deploying: 3,
  network: 4,
  finalizing: 5,
};

function firstStepError(data: NonNullable<StepsData>): string | undefined {
  return (Object.keys(STEP_ORDER) as StepKey[])
    .sort((a, b) => STEP_ORDER[a] - STEP_ORDER[b])
    .map((key) => data[key]?.error)
    .find((error) => error != null);
}

// Note: user-cancelled deployments are routed to DeploymentCancelled in
// page.tsx before this banner is rendered, so we don't need special-case
// handling for the "Cancelled by user" marker here.
export function FailedDeploymentBanner({
  stepsData,
  settingsUrl,
  limitsUrl,
  deployment,
}: {
  stepsData: StepsData | undefined;
  settingsUrl: Route;
  limitsUrl: Route;
  deployment: Deployment;
}) {
  const [redeployOpen, setRedeployOpen] = useState(false);
  const errorMessage = (stepsData && firstStepError(stepsData)) ?? "Deployment failed";
  const showSettingsLink = isSettingsRelatedError(errorMessage);
  const billingUpgrades = useBillingUIUpgrades();
  const limit = billingUpgrades ? limitFailure(errorMessage) : null;
  const actions = stepsData == null ? "loading" : limit ? "limit" : "generic";

  return (
    <div className="animate-fade-slide-in">
      <AlertBanner variant="error">
        <AlertBannerTitle>Deployment failed</AlertBannerTitle>
        <AlertBannerDescription className="max-w-200 break-words">
          {limit ?? errorMessage}
          {limit && (
            <>
              {" "}
              <Link href={LIMITS_DOCS_URL} target="_blank" rel="noopener noreferrer">
                Learn more
              </Link>
            </>
          )}
          {showSettingsLink && (
            <>
              {" "}
              <Link href={settingsUrl}>Go to Settings</Link>
            </>
          )}
        </AlertBannerDescription>
        <AlertBannerActions>
          {match(actions)
            .with("loading", () => null)
            .with("limit", () => (
              <>
                <RedeployButton variant="outline" onClick={() => setRedeployOpen(true)} />
                <Button
                  variant="primary"
                  size="sm"
                  className="px-3"
                  render={<Link href={limitsUrl} />}
                >
                  View limits
                </Button>
              </>
            ))
            .with("generic", () => <RedeployButton onClick={() => setRedeployOpen(true)} />)
            .exhaustive()}
        </AlertBannerActions>
      </AlertBanner>
      <RedeployDialog
        isOpen={redeployOpen}
        onClose={() => setRedeployOpen(false)}
        selectedDeployment={deployment}
      />
    </div>
  );
}

function RedeployButton({
  onClick,
  variant = "primary",
}: {
  onClick: () => void;
  variant?: "primary" | "outline";
}): ReactNode {
  return (
    <Button variant={variant} size="sm" onClick={onClick} className="px-3">
      Redeploy
    </Button>
  );
}
