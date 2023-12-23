import { ApiKeyTable } from "@/components/dashboard/api-key-table";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { notFound } from "next/navigation";

export const dynamic = "force-dynamic";
export const runtime = "edge";
export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.apiId), isNull(table.deletedAt)),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return notFound();
  }
  const keys = await db.query.keys.findMany({
    where: and(eq(schema.keys.keyAuthId, api.keyAuthId!), isNull(schema.keys.deletedAt)),
    limit: 100,
  });

  return (
    <div>
      <ApiKeyTable data={keys} />
    </div>
  );
}
