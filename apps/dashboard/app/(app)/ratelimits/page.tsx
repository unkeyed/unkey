import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";

import { RatelimitClient } from "./_components/ratelimit-client";
import { Navigation } from "./navigation";

export const dynamic = "force-dynamic";

export default async function RatelimitOverviewPage() {
  const tenantId = await getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
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

  return (
    <div>
      <Navigation />
      <RatelimitClient ratelimitNamespaces={workspace.ratelimitNamespaces} />
    </div>
  );
}
