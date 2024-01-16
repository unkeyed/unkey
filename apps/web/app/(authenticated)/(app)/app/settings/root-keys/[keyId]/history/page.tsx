import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { getLatestVerifications } from "@/lib/tinybird";
import { notFound } from "next/navigation";
import { AccessTable } from "./access-table";

export const runtime = "edge";

export default async function HistoryPage(props: {
  params: { keyId: string };
}) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return notFound();
  }

  const key = await db.query.keys.findFirst({
    where: eq(schema.keys.forWorkspaceId, workspace.id) && eq(schema.keys.id, props.params.keyId),
    with: {
      keyAuth: {
        with: {
          api: true,
        },
      },
    },
  });
  if (!key?.keyAuth?.api) {
    return notFound();
  }

  const history = await getLatestVerifications({
    workspaceId: workspace.id,
    apiId: key.keyAuth.api.id,
    keyId: key.id,
  });

  return <AccessTable verifications={history.data} />;
}
