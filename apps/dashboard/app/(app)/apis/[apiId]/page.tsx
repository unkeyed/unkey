import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull } from "@/lib/db";
import { apis } from "@unkey/db/src/schema";
import { redirect } from "next/navigation";
import { LogsClient } from "./_overview/logs-client";
import { ApisNavbar } from "./api-id-navbar";
export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();
  const apiId = props.params.apiId;

  const currentApi = await db.query.apis.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAtM)),
    with: {
      workspace: {
        columns: {
          id: true,
          tenantId: true,
        },
      },
    },
  });

  if (!currentApi || currentApi.workspace.tenantId !== tenantId || !currentApi?.keyAuthId) {
    return redirect("/new");
  }

  const workspaceApis = await db
    .select({
      id: apis.id,
      name: apis.name,
    })
    .from(apis)
    .where(and(eq(apis.workspaceId, currentApi.workspaceId), isNull(apis.deletedAtM)))
    .orderBy(apis.name);

  return (
    <div>
      <ApisNavbar
        api={currentApi}
        activePage={{
          href: `/apis/${apiId}`,
          text: "Requests",
        }}
        apis={workspaceApis}
      />
      <LogsClient apiId={apiId} />
    </div>
  );
}
