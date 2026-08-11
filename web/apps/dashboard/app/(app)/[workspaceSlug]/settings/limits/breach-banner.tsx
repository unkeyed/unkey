"use client";

import { AlertBanner, AlertBannerDescription, AlertBannerTitle } from "@unkey/ui";
import Link from "next/link";
import type { GroupKey } from "./limit-groups";

const SUPPORT_MAILTO = "mailto:support@unkey.com";

/** `ask` runs into the contact link, so it ends on a comma. */
const COMPUTE_ADVICE = {
  lead: "Scale down or remove a deployment to free capacity.",
  ask: "To request higher capacity limits,",
};
const API_ADVICE = {
  lead: null,
  ask: "To request a higher monthly operations limit,",
};
const MIXED_ADVICE = {
  lead: "Scale down or remove a deployment to free compute capacity.",
  ask: "To request higher limits,",
};

function advice(breached: GroupKey[]) {
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
  const { lead, ask } = advice(breached);

  return (
    <AlertBanner variant="error">
      <AlertBannerTitle>You've reached a limit</AlertBannerTitle>
      <AlertBannerDescription>
        {lead ? `${lead} ` : null}
        {ask}{" "}
        <Link href={SUPPORT_MAILTO} className="underline underline-offset-2 hover:opacity-80">
          contact us
        </Link>
        .
      </AlertBannerDescription>
    </AlertBanner>
  );
}
