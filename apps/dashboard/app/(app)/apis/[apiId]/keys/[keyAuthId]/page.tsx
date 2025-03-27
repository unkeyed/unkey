import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { Navigation } from "./navigation";
import { VirtualKeys } from "./virtual-keys";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const tenantId = getTenantId();
  const keyAuth = await db.query.keyAuth.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.keyAuthId), isNull(table.deletedAtM)),
    with: {
      workspace: true,
      api: true,
    },
  });

  if (!keyAuth || keyAuth.workspace.tenantId !== tenantId) {
    return notFound();
  }

  // Fetch keys data
  const keys = await db.query.keys.findMany({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.keyAuthId, props.params.keyAuthId), isNull(table.deletedAtM)),
    limit: 100,
    with: {
      identity: {
        columns: {
          externalId: true,
        },
      },
      roles: {
        with: {
          role: {
            with: {
              permissions: true,
            },
          },
        },
      },
      permissions: true,
    },
  });

  // Transform the data for the client component
  const transformedKeys = keys.map((key) => {
    const permissions = new Set(key.permissions.map((p) => p.permissionId));
    for (const role of key.roles) {
      for (const permission of role.role.permissions) {
        permissions.add(permission.permissionId);
      }
    }

    return {
      id: key.id,
      keyAuthId: key.keyAuthId,
      name: key.name,
      start: key.start,
      roles: key.roles.length,
      enabled: key.enabled,
      permissions: permissions.size,
      environment: key.environment,
      externalId: key.identity?.externalId ?? key.ownerId ?? null,
    };
  });

  return (
    <div>
      <Navigation apiId={props.params.apiId} keyA={keyAuth} />
      <VirtualKeys
        keys={transformedKeys}
        apiId={props.params.apiId}
        keyAuthId={props.params.keyAuthId}
      />
    </div>
  );
}
