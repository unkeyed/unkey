import { getTenantId } from "@/lib/auth";
import { type Permission, db, eq, schema } from "@/lib/db";
import { notFound } from "next/navigation";
import { Api } from "./api";
import { Legacy } from "./legacy";
import { Workspace } from "./workspace";
import { env } from "@/lib/env";
import { getLatestVerifications } from "@/lib/tinybird";
import { AccessTable } from "../history/access-table";
import { Card, CardContent, CardHeader, CardTitle, MetricCardTitle } from "@/components/ui/card";
import ms from "ms";
import { Suspense } from "react";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RootKeyPage(props: {
  params: { keyId: string };
}) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: {
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });
  if (!workspace) {
    return notFound();
  }

  const key = await db.query.keys.findFirst({
    where: eq(schema.keys.forWorkspaceId, workspace.id) && eq(schema.keys.id, props.params.keyId),
    with: {
      permissions: {
        with: {
          permission: true,
        },
      },
    },
  });
  if (!key) {
    return notFound();
  }

  const permissions = key.permissions.map((kp) => kp.permission);

  const permissionsByApi = permissions.reduce(
    (acc, permission) => {
      if (!permission.name.startsWith("api.")) {
        return acc;
      }
      const [_, apiId, _action] = permission.name.split(".");

      if (!acc[apiId]) {
        acc[apiId] = [];
      }
      acc[apiId].push(permission);
      return acc;
    },
    {} as { [apiId: string]: Permission[] },
  );

  const { UNKEY_WORKSPACE_ID, UNKEY_API_ID } = env();

  const keyForHistory = await db.query.keys.findFirst({
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
  if (!keyForHistory?.keyAuth?.api) {
    return notFound();
  }
  const history = await getLatestVerifications({
    workspaceId: UNKEY_WORKSPACE_ID,
    apiId: UNKEY_API_ID,
    keyId: key.id,
  });

  return (
    <div className="flex flex-col gap-4">
      {permissions.some((p) => p.name === "*") ? (
        <Legacy keyId={key.id} permissions={permissions} />
      ) : null}

      <Workspace keyId={key.id} permissions={permissions} />

      {/* TODO: Filter only APIs that have >= 1 active permissions */}
      {workspace.apis.map((api) => (
        <Api key={api.id} api={api} keyId={key.id} permissions={permissionsByApi[api.id] || []} />
      ))}
      {/* TODO: Add a card to trigger a Dialog for adding permissions for another API */}

      <UsageHistoryCard
        accessTableProps={{
          verifications: history.data,
        }}
      />
    </div>
  );
}

function UsageHistoryCard(props: { accessTableProps: React.ComponentProps<typeof AccessTable> }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Usage History</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-wrap justify-between divide-x [&>div:first-child]:pl-0">
        <AccessTable {...props.accessTableProps} />
      </CardContent>
    </Card>
  );
}
