import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";

export default async function SemanticCachePage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      llmGateways: {
        columns: {
          subdomain: true,
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  if (!workspace.llmGateways.length) {
    return redirect("/semantic-cache/new");
  }

  return redirect("/semantic-cache/logs");
}
