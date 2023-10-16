import { Banner } from "@/components/banner";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import ms from "ms";
import Link from "next/link";
/**
 * Shows a banner if necessary
 */
export const UsageBanner: React.FC = async () => {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: true,
    },
  });
  if (!workspace) {
    return null;
  }

  const fmt = new Intl.NumberFormat("en-US").format;

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
          <Link href="/app/stripe" className="underline">
            Add a payment method
          </Link>
        </p>
      </Banner>
    );
  }

  if (
    workspace.maxActiveKeys &&
    workspace.usageActiveKeys &&
    (workspace.usageActiveKeys ?? 0) >= workspace.maxActiveKeys
  ) {
    return (
      <Banner variant="alert">
        <p className="text-xs text-center">
          You have exceeded your plan&apos;s monthly usage limit for active keys:{" "}
          <strong>{fmt(workspace.usageActiveKeys)}</strong> /{" "}
          <strong>{fmt(workspace.maxActiveKeys)}</strong>.{" "}
          <Link href="/app/stripe" className="underline">
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

  if (
    workspace.maxVerifications &&
    workspace.usageVerifications &&
    (workspace.usageVerifications ?? 0) >= workspace.maxVerifications
  ) {
    return (
      <Banner variant="alert">
        <p className="text-xs text-center">
          You have exceeded your plan&apos;s monthly usage limit for verifications:{" "}
          <strong>{fmt(workspace.usageVerifications)}</strong> /{" "}
          <strong>{fmt(workspace.maxVerifications)}</strong>.{" "}
          <Link href="/app/stripe" className="underline">
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

  return null;
};
