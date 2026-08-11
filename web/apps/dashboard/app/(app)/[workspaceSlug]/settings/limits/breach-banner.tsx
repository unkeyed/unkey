"use client";

import { AlertBanner, AlertBannerDescription, AlertBannerTitle } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type { GroupKey } from "./limit-groups";

const SUPPORT_MAILTO = "mailto:support@unkey.com";

/**
 * The two products fail differently, so they route differently. The API
 * operations ceiling rises with the plan (`updateSubscription` writes it from the
 * Stripe product's `requestsPerMonth`), so that breach goes to Billing. The three
 * compute ceilings are identical on every plan (`lib/limits.ts`), so no upgrade
 * clears them and the only route is asking us.
 *
 * `ask` runs into the link, so it ends on a comma.
 */
const COMPUTE_ADVICE = {
  lead: "Scale down or remove a deployment to free capacity.",
  ask: "To request higher capacity limits,",
  linkLabel: "contact us",
  linkTo: "support",
} as const;
const API_ADVICE = {
  lead: null,
  ask: "You're over your plans allowed usage, to continue - please",
  linkLabel: "upgrade your plan",
  linkTo: "billing",
} as const;
const MIXED_ADVICE = {
  lead: "Scale down or remove a deployment to free compute capacity.",
  ask: "To raise your API operations limit,",
  linkLabel: "upgrade your plan",
  linkTo: "billing",
} as const;

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
export function BreachBanner({
  breached,
  billingHref,
}: {
  breached: GroupKey[];
  billingHref: Route;
}) {
  const { lead, ask, linkLabel, linkTo } = advice(breached);

  return (
    <AlertBanner variant="error">
      <AlertBannerTitle>You've reached a limit</AlertBannerTitle>
      <AlertBannerDescription>
        {lead ? `${lead} ` : null}
        {ask}{" "}
        <Link
          href={linkTo === "billing" ? billingHref : SUPPORT_MAILTO}
          className="underline underline-offset-2 hover:opacity-80"
        >
          {linkLabel}
        </Link>
        .
      </AlertBannerDescription>
    </AlertBanner>
  );
}
