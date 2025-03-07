import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { LogsClient } from "./_overview/logs-client";
import { Navigation } from "./navigation";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.apiId), isNull(table.deletedAtM)),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId || !api?.keyAuthId) {
    return redirect("/new");
  }

  return (
    <div>
      <Navigation api={api} />
      <LogsClient apiId={api.id} />
    </div>
  );
}
