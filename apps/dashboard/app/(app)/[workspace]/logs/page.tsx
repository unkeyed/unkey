import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { Layers3 } from "@unkey/icons";
import { redirect } from "next/navigation";
import { LogsClient } from "./components/logs-client";

import { Navigation } from "@/components/navigation/navigation";
export const dynamic = "force-dynamic";

export default async function Page({ params }: { params: { workspaceId: string } }) {
  const { orgId } = await getAuth();
  const workspaceId = params.workspaceId;
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return redirect("/new");
  }

  if (workspaceId !== workspace.id) {
    redirect(`/${workspace.id}/logs`);
  }

  return (
    <div>
      <Navigation href={`/${workspace.id}/logs`} name="Logs" icon={<Layers3 />} />
      <LogsClient />
    </div>
  );
}
