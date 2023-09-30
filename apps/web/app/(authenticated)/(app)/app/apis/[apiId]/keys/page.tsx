import { ApiKeyTable } from "@/components/dashboard/api-key-table";
import { getTenantId } from "@/lib/auth";
import { type Key, db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";

export const revalidate = 0;
export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db().query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return redirect("/onboarding");
  }
  const allKeys = await db().query.keys.findMany({
    where: eq(schema.keys.keyAuthId, api.keyAuthId!),
  });

  const keys: Key[] = [];
  const expired: Key[] = [];

  for (const k of allKeys) {
    if (k.expires && k.expires.getTime() < Date.now()) {
      expired.push(k);
    } else {
      keys.push(k);
    }
  }
  if (expired.length > 0) {
    await Promise.all(expired.map((k) => db().delete(schema.keys).where(eq(schema.keys.id, k.id))));
  }

  return (
    <div>
      <ApiKeyTable data={keys} />
    </div>
  );
}
