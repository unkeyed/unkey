import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { notFound } from "next/navigation";
import { AccessTable } from "./access-table";

export const runtime = "edge";

export default async function HistoryPage(props: {
  params: { keyId: string };
}) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
  });
  if (!workspace) {
    return notFound();
  }
  const { UNKEY_WORKSPACE_ID } = env();

  const key = await db.query.keys.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(
        eq(table.workspaceId, UNKEY_WORKSPACE_ID),
        eq(table.forWorkspaceId, workspace.id),
        eq(table.id, props.params.keyId),
        isNull(table.deletedAt),
      ),
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
  const history = await clickhouse.verifications.logs({
    workspaceId: UNKEY_WORKSPACE_ID,
    keySpaceId: key.keyAuthId,
    keyId: key.id,
  });

  return <AccessTable verifications={history} />;
}
