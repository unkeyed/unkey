import { PageHeader } from "@/components/dashboard/page-header";
import { RootKeyTable } from "@/components/dashboard/root-key-table";
import { Button } from "@/components/ui/button";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import Link from "next/link";
import { redirect } from "next/navigation";

export const revalidate = 0;

export default async function SettingsKeysPage(_props: {
  params: { apiId: string };
}) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      apis: true,
    },
  });
  if (!workspace) {
    return redirect("/new");
  }

  const keys = await db.query.keys.findMany({
    where: (table, { eq, and, or, isNull, gt }) =>
      and(
        eq(table.forWorkspaceId, workspace.id),
        isNull(table.deletedAt),
        or(isNull(table.expires), gt(table.expires, new Date())),
      ),
    limit: 100,
  });

  return (
    <div className="min-h-screen ">
      <PageHeader
        title="Root Keys"
        description="Root keys are used to interact with the Unkey API."
        actions={[
          <Link key="create-root-key" href="/app/settings/root-keys/new">
            <Button variant="primary">Create New Root Key</Button>
          </Link>,
        ]}
      />
      <div className="mb-20 grid w-full grid-cols-1 gap-8">
        <RootKeyTable data={keys} />
      </div>
    </div>
  );
}
