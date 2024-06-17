import { Button } from "@/components/ui/button";

import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { getAllSemanticCacheLogs } from "@/lib/tinybird";
import { redirect } from "next/navigation";
import Table from "./table";

export default async function SemanticCacheLogsPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      llmGateways: {
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  const gatewayId = workspace?.llmGateways[0]?.id;

  if (!gatewayId) {
    return redirect("/semantic-cache/new");
  }

  const { data } = await getAllSemanticCacheLogs({
    gatewayId,
    workspaceId: workspace.id,
    limit: 1000,
  });

  return <Table data={data} workspace={workspace} />;
}
