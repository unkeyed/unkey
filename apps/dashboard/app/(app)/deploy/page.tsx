import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { BranchItem } from "./client";

export const revalidate = 0;

export default async function Page() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return <div>Workspace with tenantId: {tenantId} not found</div>;
  }

  const gateway = await db.query.gateways.findFirst({
    with: {
      branches: true,
    },
  });
  if (!gateway) {
    return null;
  }
  const main = gateway.branches.find((b) => b.name === "main")!;
  return (
    <div>
      <BranchItem
        branch={{
          id: main.id,
          name: main?.name,
          domain: main?.domain,
          isDefault: true,
          children: gateway.branches
            .filter((b) => b.name !== "main")
            .map((b) => ({
              id: b.id,
              name: b.name,
              domain: `${b.domain}.unkey.app`,
              isDefault: false,
            })),
        }}
      />
    </div>
  );
}
