import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
} from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type { StepsData } from "../use-deployment-status";
import { LIMITS_DOCS_URL, limitFailure } from "./limit-failure";

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

export function FailedDeploymentBanner({
  stepsData,
  limitsUrl,
}: {
  stepsData: StepsData | undefined;
  limitsUrl: Route;
}) {
  const errorMessage = stepsData ? firstStepError(stepsData) : undefined;
  const billingUpgrades = useBillingUIUpgrades();
  const limit = billingUpgrades && errorMessage ? limitFailure(errorMessage) : null;

  return (
    <AlertBanner variant="error">
      <AlertBannerTitle>Deployment failed</AlertBannerTitle>
      {limit && (
        <>
          <AlertBannerDescription className="max-w-200 break-words">
            {limit}{" "}
            <Link href={LIMITS_DOCS_URL} target="_blank" rel="noopener noreferrer">
              Learn more
            </Link>
          </AlertBannerDescription>
          <AlertBannerActions>
            <Button variant="primary" size="sm" className="px-3" render={<Link href={limitsUrl} />}>
              View limits
            </Button>
          </AlertBannerActions>
        </>
      )}
    </AlertBanner>
  );
}
