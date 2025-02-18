import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { LogsClient } from "./_overview/logs-client";
import { NamespaceNavbar } from "./namespace-navbar";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RatelimitNamespacePage(props: {
  params: { namespaceId: string };
  searchParams: {
    identifier?: string;
  };
}) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    columns: {
      name: true,
      tenantId: true,
    },
    with: {
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAt),
        columns: {
          id: true,
          workspaceId: true,
          name: true,
        },
      },
    },
  });

  const namespace = workspace?.ratelimitNamespaces.find((r) => r.id === props.params.namespaceId);

  if (!namespace || !workspace || workspace.tenantId !== tenantId) {
    return redirect("/ratelimits");
  }

  return (
    <div>
      <NamespaceNavbar namespace={namespace} ratelimitNamespaces={workspace.ratelimitNamespaces} />
      <LogsClient namespaceId={namespace.id} />
    </div>
  );
}
