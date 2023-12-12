import { Banner } from "@/components/banner";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { activeKeys, verifications } from "@/lib/tinybird";
import ms from "ms";
import Link from "next/link";
/**
 * Shows a banner if necessary
 */
export const UsageBanner: React.FC = async () => {
  const tenantId = getTenantId();

  const freeTierQuota = {
    activeKeys: 100,
    verifications: 2500,
  };

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: true,
    },
  });
  if (!workspace) {
    return null;
  }

  const t = new Date();

  const year = t.getUTCFullYear();
  const month = t.getUTCMonth() + 1;

  const fmt = new Intl.NumberFormat("en-US").format;

  if (workspace.plan === "free") {
    const [usedActiveKeys, usedVerifications] = await Promise.all([
      activeKeys({
        workspaceId: workspace.id,
        year,
        month,
      }).then((res) => res.data.at(0)?.keys ?? 0),
      verifications({
        workspaceId: workspace.id,
        year,
        month,
      }).then((res) => res.data.at(0)?.success ?? 0),
    ]);

    if (usedActiveKeys >= freeTierQuota.activeKeys) {
      return (
        <Banner variant="alert">
          <p className="text-xs text-center">
            You have exceeded your plan&apos;s monthly usage limit for active keys:{" "}
            <strong>{fmt(usedActiveKeys)}</strong> /{" "}
            <strong>{fmt(freeTierQuota.activeKeys)}</strong>.{" "}
            <Link href="/app/settings/billing/stripe" className="underline">
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

    if (usedVerifications >= freeTierQuota.verifications) {
      return (
        <Banner variant="alert">
          <p className="text-xs text-center">
            You have exceeded your plan&apos;s monthly usage limit for verifications:{" "}
            <strong>{fmt(usedVerifications)}</strong> /{" "}
            <strong>{fmt(freeTierQuota.verifications)}</strong>.{" "}
            <Link href="/app/settings/billing/stripe" className="underline">
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
          <Link href="/app/settings/billing/stripe" className="underline">
            Add a payment method
          </Link>
        </p>
      </Banner>
    );
  }

  return null;
};
