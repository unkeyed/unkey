"use client";

import { usePayInvoice, usePaymentIssues } from "@/components/billing/payment-issues";
import { routes } from "@/lib/navigation/routes";
import type { SubscriptionProduct } from "@/lib/stripe/billingSubscriptions";
import { useWorkspace } from "@/providers/workspace-provider";
import { match } from "@unkey/match";
import Link from "next/link";
import { AppBanner } from "./app-banner";

type BannerAction = { kind: "pay"; product: SubscriptionProduct } | { kind: "billing" };

function bannerAction(issues: SubscriptionProduct[], isAdmin: boolean): BannerAction {
  const onlyIssue = issues.length === 1 ? issues[0] : undefined;
  return isAdmin && onlyIssue ? { kind: "pay", product: onlyIssue } : { kind: "billing" };
}

export function PaymentFailedBanner() {
  const { workspace, user } = useWorkspace();
  const issues = usePaymentIssues();
  const payInvoice = usePayInvoice();

  if (!workspace || issues.length === 0) {
    return null;
  }

  return (
    <AppBanner
      tone="error"
      action={match(bannerAction(issues, user?.role === "admin"))
        .with({ kind: "pay" }, ({ product }) => (
          <button
            type="button"
            aria-busy={payInvoice.busy}
            disabled={payInvoice.busy}
            onClick={() => payInvoice.pay(product)}
          >
            {payInvoice.busy ? "Opening invoice…" : "Pay invoice"}
          </button>
        ))
        .with({ kind: "billing" }, () => (
          <Link href={routes.settings.billing({ workspaceSlug: workspace.slug })}>
            Go to billing
          </Link>
        ))
        .exhaustive()}
    >
      <span className="font-semibold">Payment failed.</span> Your last payment did not go through.
    </AppBanner>
  );
}
