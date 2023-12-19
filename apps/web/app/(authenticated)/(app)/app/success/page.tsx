import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { getQ1ActiveWorkspaces } from "@/lib/tinybird";
import { notFound } from "next/navigation";

export default async function SuccessPage() {
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, getTenantId()), isNull(table.deletedAt)),
  });
  if (!workspace?.features.successPage) {
    return notFound();
  }

  const activeWorkspaces = await getQ1ActiveWorkspaces({});

  return <div>{JSON.stringify(activeWorkspaces.data, null, 2)}</div>;
}
