import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Secrets } from "./secrets";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function SecretsPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq, isNull, and }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      secrets: true,
    },
  });
  if (!workspace) {
    return notFound();
  }

  return (
    <>
      <PageHeader
        title="Workspace Secrets"
        description="All of your secrets are here"
        actions={[
          <Link key="new" href="/secrets/new">
            <Button>Create secret</Button>
          </Link>,
        ]}
      />
      <Secrets secrets={workspace.secrets} />
    </>
  );
}
