"use client";

import { AlertBanner, AlertBannerDescription, AlertBannerTitle } from "@unkey/ui";
import type { GroupKey } from "./limit-groups";

const COMPUTE_ADVICE =
  "Scale down or remove a deployment to free capacity, or contact us about a higher limit.";
const API_ADVICE = "Contact us to raise your monthly operations limit.";
const MIXED_ADVICE =
  "Scale down or remove a deployment to free compute capacity, and contact us about your other limits.";

function advice(breached: GroupKey[]): string {
  const compute = breached.includes("compute");
  const api = breached.includes("api");
  if (compute && api) {
    return MIXED_ADVICE;
  }
  return compute ? COMPUTE_ADVICE : API_ADVICE;
}

/**
 * Only rendered once a ceiling is reached. It names neither the limit nor the
 * consequence: the rows below carry the badge and the figures, and some of
 * these ceilings reject a deploy while others only send an email.
 */
export function BreachBanner({ breached }: { breached: GroupKey[] }) {
  return (
    <AlertBanner variant="error">
      <AlertBannerTitle>You've reached a limit</AlertBannerTitle>
      <AlertBannerDescription>{advice(breached)}</AlertBannerDescription>
    </AlertBanner>
  );
}
