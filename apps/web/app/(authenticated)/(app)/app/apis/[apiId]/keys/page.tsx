import { ApiKeyTable } from "@/components/dashboard/api-key-table";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { redirect } from "next/navigation";

export const revalidate = 0;
export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return redirect("/onboarding");
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
