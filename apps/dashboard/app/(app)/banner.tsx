import { Banner } from "@/components/banner";
import type { Workspace } from "@/lib/db";
import { verifications } from "@/lib/tinybird";
import { QUOTA } from "@unkey/billing";
import ms from "ms";
import Link from "next/link";

/**
 * Shows a banner if necessary
 */
export const UsageBanner: React.FC<{ workspace: Workspace | undefined }> = async ({
  workspace,
}) => {
  if (!workspace) {
    return null;
  }

  const t = new Date();

  const year = t.getUTCFullYear();
  const month = t.getUTCMonth() + 1;

  const fmt = new Intl.NumberFormat("en-US").format;

  if (workspace.plan === "free") {
    const [usedVerifications] = await Promise.all([
      verifications({
        workspaceId: workspace.id,
        year,
        month,
      }).then((res) => res.data.at(0)?.success ?? 0),
    ]);

    if (usedVerifications >= QUOTA.free.maxVerifications) {
      return (
        <Banner variant="alert">
          <p className="text-xs text-center">
            You have exceeded your plan&apos;s monthly usage limit for verifications:{" "}
            <strong>{fmt(usedVerifications)}</strong> /{" "}
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
