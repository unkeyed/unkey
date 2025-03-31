import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Layers3 } from "@unkey/icons";
import { notFound } from "next/navigation";
import { LogsClient } from "./components/logs-client";

import { Navigation } from "@/components/navigation/navigation";
export const dynamic = "force-dynamic";

export default async function Page() {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return notFound();
  }

  return (
    <div>
      <Navigation href="/logs" name="Logs" icon={<Layers3 />} />
      <LogsClient />
    </div>
  );
}
