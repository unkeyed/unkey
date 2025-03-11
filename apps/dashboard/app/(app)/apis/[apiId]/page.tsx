import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { LogsClient } from "./_overview/logs-client";
import { ApisNavbar } from "./api-id-navbar";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const apis = await db.query.apis.findMany({
    where: (table, { and, isNull }) => and(isNull(table.deletedAtM)),
    with: {
      workspace: true,
    },
  });

  const api = apis.find((api) => props.params.apiId === api.id);

  if (!api || api.workspace.tenantId !== tenantId || !api?.keyAuthId) {
    return redirect("/new");
  }

  return (
    <div>
      <ApisNavbar
        api={api}
        activePage={{
          href: `/apis/${api.id}`,
          text: "Requests",
        }}
        apis={apis.map((api) => ({ name: api.name, id: api.id }))}
      />
      <LogsClient apiId={api.id} />
    </div>
  );
}
