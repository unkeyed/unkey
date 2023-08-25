import { Banner } from "@/components/banner";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
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

  // if (team.trialExpires) {
  //   return (
  //     <Banner>
  //       Your trial expires in {ms(team.trialExpires.getTime() - Date.now(), { long: true })}.{" "}
  //       <Link href={`/${team.slug}/settings/usage`} className="underline">
  //         Add a payment method
  //       </Link>{" "}
  //       to keep using Planetfall.
  //     </Banner>
  //   );
  // }

  if (
    workspace.maxActiveKeys &&
    workspace.usageActiveKeys &&
    (workspace.usageActiveKeys ?? 0) >= workspace.maxActiveKeys
  ) {
    return (
      <Banner variant="alert">
        <div className="text-center">
          <div>
            You have exceeded your plan&apos;s monthly usage limit for active keys:{" "}
            <strong>{fmt(workspace.usageActiveKeys)}</strong> /{" "}
            <strong>{fmt(workspace.maxActiveKeys)}</strong>
          </div>
          <div>
            <Link href="/app/stripe" className="underline">
              Upgrade your plan
            </Link>{" "}
            or{" "}
            <Link href="mailto:andreas@unkey.dev" className="underline">
              contact us.
            </Link>
          </div>
        </div>
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
        <div className="text-center">
          <div>
            You have exceeded your plan&apos;s monthly usage limit for verifications:{" "}
            <strong>{fmt(workspace.usageVerifications)}</strong> /{" "}
            <strong>{fmt(workspace.maxVerifications)}</strong>
          </div>
          <div>
            <Link href="/app/stripe" className="underline">
              Upgrade your plan
            </Link>{" "}
            or{" "}
            <Link href="mailto:andreas@unkey.dev" className="underline">
              contact us.
            </Link>
          </div>
        </div>
      </Banner>
    );
  }

  return null;
};
