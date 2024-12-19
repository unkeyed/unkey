import { PageContent } from "@/components/page-content";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { DialogTrigger } from "@/components/ui/dialog";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { type Permission, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Button } from "@unkey/ui";
import { notFound } from "next/navigation";
import { AccessTable } from "./history/access-table";
import { Navigation } from "./navigation";
import { PageLayout } from "./page-layout";
import { DialogAddPermissionsForAPI } from "./permissions/add-permission-for-api";
import { Api } from "./permissions/api";
import { Legacy } from "./permissions/legacy";
import { apiPermissions } from "./permissions/permissions";
import { Workspace } from "./permissions/workspace";
import { UpdateRootKeyName } from "./update-root-key-name";

export const dynamic = "force-dynamic";

export default async function RootKeyPage(props: {
  params: { keyId: string };
}) {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
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

  const { UNKEY_WORKSPACE_ID } = env();

  const keyForHistory = await db.query.keys.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(
        eq(table.workspaceId, UNKEY_WORKSPACE_ID),
        eq(table.forWorkspaceId, workspace.id),
        eq(table.id, props.params.keyId),
        isNull(table.deletedAtM),
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
  const history = await clickhouse.verifications
    .latest({
      workspaceId: UNKEY_WORKSPACE_ID,
      keySpaceId: key.keyAuthId,
      keyId: key.id,
    })
    .then((res) => res.val!);

  const apis = workspace.apis.map((api) => {
    const apiPermissionsStructure = apiPermissions(api.id);
    const hasActivePermissions = Object.entries(apiPermissionsStructure).some(
      ([_category, allPermissions]) => {
        const amountActivePermissions = Object.entries(allPermissions).filter(
          ([_action, { description: _description, permission }]) => {
            return permissions.some((p) => p.name === permission);
          },
        );

        return amountActivePermissions.length > 0;
      },
    );

    return {
      ...api,
      hasActivePermissions,
    };
  });

  const apisWithActivePermissions = apis.filter((api) => api.hasActivePermissions);
  const apisWithoutActivePermissions = apis.filter((api) => !api.hasActivePermissions);

  return (
    <div>
      <Navigation keyId={key.id} />
      <PageContent>
        <PageLayout params={{ keyId: key.id }} rootKey={key}>
          <div className="flex flex-col gap-4">
            {permissions.some((p) => p.name === "*") ? (
              <Legacy keyId={key.id} permissions={permissions} />
            ) : null}

            <UpdateRootKeyName apiKey={key} />

            <Workspace keyId={key.id} permissions={permissions} />

            {apisWithActivePermissions.map((api) => (
              <Api
                key={api.id}
                api={api}
                keyId={key.id}
                permissions={permissionsByApi[api.id] || []}
              />
            ))}

            <DialogAddPermissionsForAPI
              keyId={props.params.keyId}
              apis={workspace.apis}
              permissions={permissions}
            >
              {apisWithoutActivePermissions.length > 0 && (
                <Card className="flex w-full items-center justify-center h-36 border-dashed">
                  <DialogTrigger asChild>
                    <Button>
                      Add permissions for {apisWithActivePermissions.length > 0 ? "another" : "an"}{" "}
                      API
                    </Button>
                  </DialogTrigger>
                </Card>
              )}
            </DialogAddPermissionsForAPI>

            <UsageHistoryCard
              accessTableProps={{
                verifications: history,
              }}
            />
          </div>
        </PageLayout>
      </PageContent>
    </div>
  );
}

function UsageHistoryCard(props: {
  accessTableProps: React.ComponentProps<typeof AccessTable>;
}) {
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
