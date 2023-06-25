import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@/lib/db";
import { redirect } from "next/navigation";
import { ApiKeyTable } from "@/components/ApiKeyTable";

export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
    with: {
      workspace: true,
      keys: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return redirect("/onboarding");
  }

  return (
    <div>
      <ApiKeyTable data={api.keys} />
    </div>
  );
}
