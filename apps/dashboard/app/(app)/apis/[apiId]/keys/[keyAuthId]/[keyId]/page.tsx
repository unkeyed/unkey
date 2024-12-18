import { ArrowLeft, Settings2 } from "lucide-react";
import Link from "next/link";

import { type Interval, IntervalSelect } from "@/app/(app)/apis/[apiId]/select";
import { CreateNewPermission } from "@/app/(app)/authorization/permissions/create-new-permission";
import type { NestedPermissions } from "@/app/(app)/authorization/roles/[roleId]/tree";
import { CreateNewRole } from "@/app/(app)/authorization/roles/create-new-role";
import { StackedColumnChart } from "@/components/dashboard/charts";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Metric } from "@/components/ui/metric";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { Minus } from "lucide-react";
import ms from "ms";
import { notFound } from "next/navigation";
import { Navigation } from "./navigation";
import PermissionTree from "./permission-list";
import { VerificationTable } from "./verification-table";
export default async function APIKeyDetailPage(props: {
  params: {
    apiId: string;
    keyId: string;
    keyAuthId: string;
  };
  searchParams: {
    interval?: Interval;
  };
}) {
  const tenantId = await getTenantId();

  const key = await db.query.keys.findFirst({
    where: and(eq(schema.keys.id, props.params.keyId), isNull(schema.keys.deletedAtM)),
    with: {
      keyAuth: true,
      roles: {
        with: {
          role: {
            with: {
              permissions: {
                with: {
                  permission: true,
                },
              },
            },
          },
        },
      },
      permissions: true,
      workspace: {
        with: {
          roles: {
            with: {
              permissions: true,
            },
          },
          permissions: {
            with: {
              roles: true,
            },
          },
        },
      },
    },
  });

  if (!key || key.workspace.tenantId !== tenantId) {
    return notFound();
  }

  const api = await db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.keyAuthId, key.keyAuthId), isNull(table.deletedAtM)),
  });
  if (!api) {
    return notFound();
  }

  const interval = props.searchParams.interval ?? "7d";

  const { getVerificationsPerInterval, start, end, granularity } = prepareInterval(interval);
  const query = {
    workspaceId: api.workspaceId,
    keySpaceId: key.keyAuthId,
    keyId: key.id,
    start,
    end,
  };
  const [verifications, latestVerifications, lastUsed] = await Promise.all([
    getVerificationsPerInterval(query),
    clickhouse.verifications.logs({
      workspaceId: key.workspaceId,
      keySpaceId: key.keyAuthId,
      keyId: key.id,
    }),
    clickhouse.verifications
      .latest({
        workspaceId: key.workspaceId,
        keySpaceId: key.keyAuthId,
        keyId: key.id,
      })
      .then((res) => res.val?.at(0)?.time ?? 0),
  ]);

  // Sort all verifications by time first
  const sortedVerifications = verifications.val!.sort((a, b) => a.time - b.time);

  const successOverTime: { x: string; y: number }[] = [];
  const ratelimitedOverTime: { x: string; y: number }[] = [];
  const usageExceededOverTime: { x: string; y: number }[] = [];
  const disabledOverTime: { x: string; y: number }[] = [];
  const insufficientPermissionsOverTime: { x: string; y: number }[] = [];
  const expiredOverTime: { x: string; y: number }[] = [];
  const forbiddenOverTime: { x: string; y: number }[] = [];

  // Get all unique timestamps
  const uniqueDates = [...new Set(sortedVerifications.map((d) => d.time))].sort((a, b) => a - b);

  // Ensure each array has entries for all timestamps with zero counts
  for (const timestamp of uniqueDates) {
    const x = new Date(timestamp).toISOString();
    successOverTime.push({ x, y: 0 });
    ratelimitedOverTime.push({ x, y: 0 });
    usageExceededOverTime.push({ x, y: 0 });
    disabledOverTime.push({ x, y: 0 });
    insufficientPermissionsOverTime.push({ x, y: 0 });
    expiredOverTime.push({ x, y: 0 });
    forbiddenOverTime.push({ x, y: 0 });
  }

  for (const d of sortedVerifications) {
    const x = new Date(d.time).toISOString();
    const index = uniqueDates.indexOf(d.time);

    switch (d.outcome) {
      case "":
      case "VALID":
        successOverTime[index] = { x, y: d.count };
        break;
      case "RATE_LIMITED":
        ratelimitedOverTime[index] = { x, y: d.count };
        break;
      case "USAGE_EXCEEDED":
        usageExceededOverTime[index] = { x, y: d.count };
        break;
      case "DISABLED":
        disabledOverTime[index] = { x, y: d.count };
        break;
      case "INSUFFICIENT_PERMISSIONS":
        insufficientPermissionsOverTime[index] = { x, y: d.count };
        break;
      case "EXPIRED":
        expiredOverTime[index] = { x, y: d.count };
        break;
      case "FORBIDDEN":
        forbiddenOverTime[index] = { x, y: d.count };
        break;
    }
  }

  const verificationsData = [
    ...successOverTime.map((d) => ({
      ...d,
      category: "Successful Verifications",
    })),
    ...ratelimitedOverTime.map((d) => ({ ...d, category: "Ratelimited" })),
    ...usageExceededOverTime.map((d) => ({ ...d, category: "Usage Exceeded" })),
    ...disabledOverTime.map((d) => ({ ...d, category: "Disabled" })),
    ...insufficientPermissionsOverTime.map((d) => ({
      ...d,
      category: "Insufficient Permissions",
    })),
    ...expiredOverTime.map((d) => ({ ...d, category: "Expired" })),
    ...forbiddenOverTime.map((d) => ({ ...d, category: "Forbidden" })),
  ];

  const transientPermissionIds = new Set<string>();
  const connectedRoleIds = new Set<string>();
  for (const role of key.roles) {
    connectedRoleIds.add(role.roleId);
  }
  for (const role of key.workspace.roles) {
    if (connectedRoleIds.has(role.id)) {
      for (const p of role.permissions) {
        transientPermissionIds.add(p.permissionId);
      }
    }
  }

  const stats = {
    valid: 0,
    ratelimited: 0,
    usageExceeded: 0,
    disabled: 0,
    insufficientPermissions: 0,
    expired: 0,
    forbidden: 0,
  };
  verifications.val!.forEach((v) => {
    switch (v.outcome) {
      case "VALID":
        stats.valid += v.count;
        break;
      case "RATE_LIMITED":
        stats.ratelimited += v.count;
        break;
      case "USAGE_EXCEEDED":
        stats.usageExceeded += v.count;
        break;
      case "DISABLED":
        stats.disabled += v.count;
        break;
      case "INSUFFICIENT_PERMISSIONS":
        stats.insufficientPermissions += v.count;
        break;
      case "EXPIRED":
        stats.expired += v.count;
        break;
      case "FORBIDDEN":
        stats.forbidden += v.count;
    }
  });

  const roleTee = key.workspace.roles.map((role) => {
    const nested: NestedPermissions = {};
    for (const permission of key.workspace.permissions) {
      let n = nested;
      const parts = permission.name.split(".");
      for (let i = 0; i < parts.length; i++) {
        const p = parts[i];
        if (!(p in n)) {
          n[p] = {
            id: permission.id,
            name: permission.name,
            description: permission.description,
            checked: role.permissions.some((p) => p.permissionId === permission.id),
            part: p,
            permissions: {},
            path: parts.slice(0, i).join("."),
          };
        }
        n = n[p].permissions;
      }
    }
    const data = {
      id: role.id,
      name: role.name,
      description: role.description,
      keyId: key.id,
      active: key.roles.some((keyRole) => keyRole.roleId === role.id),
      nestedPermissions: nested,
    };
    return data;
  });

  return (
    <div>
      <Navigation api={api} apiKey={key} />

      <PageContent>
        <div className="flex flex-col">
          <div className="flex items-center justify-between w-full">
            <Link
              href={`/apis/${props.params.apiId}/keys/${props.params.keyAuthId}/`}
              className="flex w-fit items-center gap-1 text-sm duration-200 text-content-subtle hover:text-secondary-foreground"
            >
              <ArrowLeft className="w-4 h-4" /> Back to API Keys listing
            </Link>
            <Link
              href={`/apis/${props.params.apiId}/keys/${props.params.keyAuthId}/${props.params.keyId}/settings`}
            >
              <Button>
                <Settings2 />
                Key settings
              </Button>
            </Link>
          </div>

          <div className="flex flex-col gap-4 mt-4">
            <Card>
              <CardContent className="grid grid-cols-2 sm:grid-cols-3 gap-2 sm:divide-x">
                <Metric
                  label={
                    key.expires && key.expires.getTime() < Date.now() ? "Expired" : "Expires in"
                  }
                  value={key.expires ? ms(key.expires.getTime() - Date.now()) : <Minus />}
                />
                <Metric
                  label="Remaining"
                  value={
                    typeof key.remaining === "number" ? formatNumber(key.remaining) : <Minus />
                  }
                />
                <Metric
                  label="Last Used"
                  value={lastUsed ? `${ms(Date.now() - lastUsed)} ago` : <Minus />}
                />
              </CardContent>
            </Card>
            <Separator className="my-8" />

            <div className="flex items-center justify-between">
              <h2 className="text-2xl font-semibold leading-none tracking-tight">Verifications</h2>

              <div>
                <IntervalSelect defaultSelected={interval} />
              </div>
            </div>

            {verificationsData.some(({ y }) => y > 0) ? (
              <Card>
                <CardHeader>
                  <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 divide-x">
                    <Metric label="Valid" value={formatNumber(stats.valid)} />
                    <Metric label="Ratelimited" value={formatNumber(stats.ratelimited)} />
                    <Metric label="Usage Exceeded" value={formatNumber(stats.usageExceeded)} />
                    <Metric label="Disabled" value={formatNumber(stats.disabled)} />
                    <Metric
                      label="Insufficient Permissions"
                      value={formatNumber(stats.insufficientPermissions)}
                    />
                    <Metric label="Expired" value={formatNumber(stats.expired)} />
                    <Metric label="Forbidden" value={formatNumber(stats.forbidden)} />
                  </div>
                </CardHeader>
                <CardContent>
                  <StackedColumnChart
                    colors={["primary", "warn", "danger"]}
                    data={verificationsData}
                    timeGranularity={
                      granularity >= 1000 * 60 * 60 * 24 * 30
                        ? "month"
                        : granularity >= 1000 * 60 * 60 * 24
                          ? "day"
                          : "hour"
                    }
                  />
                </CardContent>
              </Card>
            ) : (
              <Empty>
                <Empty.Icon />
                <Empty.Title>Not used</Empty.Title>
                <Empty.Description>This key was not used in the last {interval}</Empty.Description>
              </Empty>
            )}

            {latestVerifications.val && latestVerifications.val.length > 0 ? (
              <>
                <Separator className="my-8" />
                <h2 className="text-2xl font-semibold leading-none tracking-tight mt-8">
                  Latest Verifications
                </h2>
                <VerificationTable verifications={latestVerifications} />
              </>
            ) : null}

            <Separator className="my-8" />
            <div className="flex w-full flex-1 items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Badge variant="secondary" className="h-8">
                  {Intl.NumberFormat().format(key.roles.length)} Roles{" "}
                </Badge>
                <Badge variant="secondary" className="h-8">
                  {Intl.NumberFormat().format(transientPermissionIds.size)} Permissions
                </Badge>
              </div>
              <div className="flex items-center gap-2 border-border">
                <CreateNewRole
                  trigger={<Button>Create New Role</Button>}
                  permissions={key.workspace.permissions}
                />
                <CreateNewPermission trigger={<Button>Create New Permission</Button>} />
              </div>
            </div>

            <PermissionTree roles={roleTee} />
          </div>
        </div>
      </PageContent>
    </div>
  );
}

function prepareInterval(interval: Interval) {
  const now = new Date();

  switch (interval) {
    case "24h": {
      const end = now.setUTCHours(now.getUTCHours() + 1, 0, 0, 0);
      const intervalMs = 1000 * 60 * 60 * 24;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60,
        getVerificationsPerInterval: clickhouse.verifications.perHour,
      };
    }
    case "7d": {
      now.setUTCDate(now.getUTCDate() + 1);
      const end = now.setUTCHours(0, 0, 0, 0);
      const intervalMs = 1000 * 60 * 60 * 24 * 7;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60 * 24,
        getVerificationsPerInterval: clickhouse.verifications.perDay,
      };
    }
    case "30d": {
      now.setUTCDate(now.getUTCDate() + 1);
      const end = now.setUTCHours(0, 0, 0, 0);
      const intervalMs = 1000 * 60 * 60 * 24 * 30;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60 * 24,
        getVerificationsPerInterval: clickhouse.verifications.perDay,
      };
    }
    case "90d": {
      now.setUTCDate(now.getUTCDate() + 1);
      const end = now.setUTCHours(0, 0, 0, 0);
      const intervalMs = 1000 * 60 * 60 * 24 * 90;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60 * 24,
        getVerificationsPerInterval: clickhouse.verifications.perDay,
      };
    }
  }
}
