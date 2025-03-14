import { Banner } from "@/components/banner";
import { clickhouse } from "@/lib/clickhouse";
import type { Workspace } from "@/lib/db";
import { formatNumber as fmt } from "@/lib/fmt";
import { QUOTA } from "@unkey/billing";
import ms from "ms";
import Link from "next/link";

/**
 * Shows a banner if necessary
 */
export const UsageBanner: React.FC<{
  workspace: Workspace | undefined;
}> = async ({ workspace }) => {
  if (!workspace) {
    return null;
  }

  const t = new Date();

  const year = t.getUTCFullYear();
  const month = t.getUTCMonth() + 1;

  if (workspace.plan === "free") {
    const billableVerifications = await clickhouse.billing.billableVerifications({
      workspaceId: workspace.id,
      year,
      month,
    });

    if (billableVerifications >= QUOTA.free.maxVerifications) {
      return (
        <Banner variant="alert">
          <p className="text-xs text-center">
            You have exceeded your plan&apos;s monthly usage limit for verifications:{" "}
            <strong>{fmt(billableVerifications)}</strong> /{" "}
            <strong>{fmt(QUOTA.free.maxVerifications)}</strong>.{" "}
            <Link href="/settings/billing" className="underline">
              Upgrade your plan
            </Link>{" "}
            or{" "}
            <Link href="mailto:support@unkey.dev" className="underline">
              contact us.
            </Link>
          </p>
        </Banner>
      );
    }
  }
  // Show a banner if their trial is ending within 7 days
  if (workspace.trialEnds && workspace.trialEnds.getTime() < Date.now() + 1000 * 60 * 60 * 24 * 7) {
    return (
      <Banner>
        <p className="text-xs text-center">
          {workspace.trialEnds.getTime() <= Date.now()
            ? "Your trial has expired."
            : `Your trial expires in ${ms(workspace.trialEnds.getTime() - Date.now(), {
                long: true,
              })}.`}{" "}
          <Link href="/settings/billing" className="underline">
            Add a payment method
          </Link>
        </p>
      </Banner>
    );
  }

  return null;
};
