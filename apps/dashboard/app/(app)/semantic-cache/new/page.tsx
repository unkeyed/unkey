import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import Form from "../form";

export default async function NewSemanticCachePage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      gateways: {
        where: (table, { isNull }) => isNull(table.deletedAt),
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (workspace?.gateways.length) {
    return redirect("/semantic-cache/logs");
  }

  return <Form />;
}
