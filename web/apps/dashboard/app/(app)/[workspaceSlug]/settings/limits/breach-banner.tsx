"use client";

import { AlertBanner, AlertBannerDescription, AlertBannerTitle } from "@unkey/ui";
import Link from "next/link";
import type { GroupKey } from "./limit-groups";

const SUPPORT_MAILTO = "mailto:support@unkey.com";

const COMPUTE_ADVICE = "Scale down or remove a deployment to free capacity.";
const API_ADVICE = "Monthly operations limits are raised on request.";
const MIXED_ADVICE = "Scale down or remove a deployment to free compute capacity.";

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
      <AlertBannerDescription>
        {advice(breached)}{" "}
        <Link href={SUPPORT_MAILTO} className="underline underline-offset-2 hover:opacity-80">
          Contact us
        </Link>
      </AlertBannerDescription>
    </AlertBanner>
  );
}
