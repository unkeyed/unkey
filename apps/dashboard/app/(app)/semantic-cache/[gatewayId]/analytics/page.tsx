import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { BarChart } from "lucide-react";
import { redirect } from "next/navigation";

export default async function SemanticCacheAnalyticsPage() {
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

  const gateway = workspace.llmGateways.at(0);

  if (!gateway) {
    return redirect("/apis");
  }

  return (
    <EmptyPlaceholder>
      <EmptyPlaceholder.Icon>
        <BarChart />
      </EmptyPlaceholder.Icon>
      <EmptyPlaceholder.Title>Semantic caching is deprecated</EmptyPlaceholder.Title>
      <EmptyPlaceholder.Description>
        This service will be removed at the end of the year.
      </EmptyPlaceholder.Description>
    </EmptyPlaceholder>
  );
}
