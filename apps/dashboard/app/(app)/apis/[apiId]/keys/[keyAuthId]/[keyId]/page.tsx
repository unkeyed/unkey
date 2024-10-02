import { ArrowLeft, Settings2 } from "lucide-react";
import Link from "next/link";

import { type Interval, IntervalSelect } from "@/app/(app)/apis/[apiId]/select";
import { CreateNewPermission } from "@/app/(app)/authorization/permissions/create-new-permission";
import { CreateNewRole } from "@/app/(app)/authorization/roles/create-new-role";
import { StackedColumnChart } from "@/components/dashboard/charts";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { serverAuth } from "@/lib/auth/server";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import {
  getLastUsed,
  getLatestVerifications,
  getVerificationsDaily,
  getVerificationsHourly,
} from "@/lib/tinybird";
import { cn } from "@/lib/utils";
import { BarChart, Minus } from "lucide-react";
import ms from "ms";
import { notFound } from "next/navigation";
import { Chart } from "./chart";
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
  const tenantId = await serverAuth.getTenantId();

  const key = await db.query.keys.findFirst({
    where: and(eq(schema.keys.id, props.params.keyId), isNull(schema.keys.deletedAt)),
    with: {
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
      and(eq(table.keyAuthId, key.keyAuthId), isNull(table.deletedAt)),
  });
  if (!api) {
    return notFound();
  }

  const interval = props.searchParams.interval ?? "7d";

  const { getVerificationsPerInterval, start, end, granularity } = prepareInterval(interval);
  const query = {
    workspaceId: api.workspaceId,
    apiId: api.id,
    keyId: key.id,
    start,
    end,
  };
  const [verifications, totalUsage, latestVerifications, lastUsed] = await Promise.all([
    getVerificationsPerInterval(query),
    getVerificationsPerInterval({
      workspaceId: api.workspaceId,
      apiId: api.id,
      keyId: key.id,
    }).then((res) => res.data.at(0) ?? { success: 0, rateLimited: 0, usageExceeded: 0 }), // no interval -> a
    getLatestVerifications({
      workspaceId: key.workspaceId,
      apiId: api.id,
      keyId: key.id,
    }),
    getLastUsed({ keyId: key.id }).then((res) => res.data.at(0)?.lastUsed ?? 0),
  ]);

  const successOverTime: { x: string; y: number }[] = [];
  const ratelimitedOverTime: { x: string; y: number }[] = [];
  const usageExceededOverTime: { x: string; y: number }[] = [];
  const disabledOverTime: { x: string; y: number }[] = [];
  const insufficientPermissionsOverTime: { x: string; y: number }[] = [];
  const expiredOverTime: { x: string; y: number }[] = [];
  const forbiddenOverTime: { x: string; y: number }[] = [];

  for (const d of verifications.data.sort((a, b) => a.time - b.time)) {
    const x = new Date(d.time).toISOString();
    successOverTime.push({ x, y: d.success });
    ratelimitedOverTime.push({ x, y: d.rateLimited });
    usageExceededOverTime.push({ x, y: d.usageExceeded });
    disabledOverTime.push({ x, y: d.disabled });
    insufficientPermissionsOverTime.push({ x, y: d.insufficientPermissions });
    expiredOverTime.push({ x, y: d.expired });
    forbiddenOverTime.push({ x, y: d.forbidden });
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

  return (
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
          className={cn(buttonVariants({ variant: "outline" }), "gap-1")}
        >
          <Settings2 className="w-4 h-4" />
          Key settings
        </Link>
      </div>

      <div className="flex flex-col gap-4 mt-4">
        <Card>
          <CardContent className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-2 sm:divide-x">
            <Metric
              label={key.expires && key.expires.getTime() < Date.now() ? "Expired" : "Expires in"}
              value={key.expires ? ms(key.expires.getTime() - Date.now()) : <Minus />}
            />
            <Metric
              label="Remaining"
              value={typeof key.remaining === "number" ? formatNumber(key.remaining) : <Minus />}
            />
            <Metric
              label="Last Used"
              value={lastUsed ? `${ms(Date.now() - lastUsed)} ago` : <Minus />}
            />
            <Metric label="Success" value={formatNumber(totalUsage.success)} />
            <Metric label="Ratelimited" value={formatNumber(totalUsage.rateLimited)} />
            <Metric label="Usage Exceeded" value={formatNumber(totalUsage.usageExceeded)} />
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
                <Metric
                  label="Valid"
                  value={formatNumber(
                    verifications.data.reduce((sum, day) => sum + day.success, 0),
                  )}
                />
                <Metric
                  label="Ratelimited"
                  value={formatNumber(
                    verifications.data.reduce((sum, day) => sum + day.rateLimited, 0),
                  )}
                />
                <Metric
                  label="Usage Exceeded"
                  value={formatNumber(
                    verifications.data.reduce((sum, day) => sum + day.usageExceeded, 0),
                  )}
                />
                <Metric
                  label="Disabled"
                  value={formatNumber(
                    verifications.data.reduce((sum, day) => sum + day.disabled, 0),
                  )}
                />
                <Metric
                  label="Insufficient Permissions"
                  value={formatNumber(
                    verifications.data.reduce((sum, day) => sum + day.insufficientPermissions, 0),
                  )}
                />
                <Metric
                  label="Expired"
                  value={formatNumber(
                    verifications.data.reduce((sum, day) => sum + day.expired, 0),
                  )}
                />
                <Metric
                  label="Forbidden"
                  value={formatNumber(
                    verifications.data.reduce((sum, day) => sum + day.forbidden, 0),
                  )}
                />
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
          <EmptyPlaceholder>
            <EmptyPlaceholder.Icon>
              <BarChart />
            </EmptyPlaceholder.Icon>
            <EmptyPlaceholder.Title>Not used</EmptyPlaceholder.Title>
            <EmptyPlaceholder.Description>
              This key was not used in the last {interval}
            </EmptyPlaceholder.Description>
          </EmptyPlaceholder>
        )}

        {latestVerifications.data.length > 0 ? (
          <>
            <Separator className="my-8" />
            <h2 className="text-2xl font-semibold leading-none tracking-tight mt-8">
              Latest Verifications
            </h2>
            <VerificationTable verifications={latestVerifications.data} />
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
          <div className="flex items-center gap-2">
            <CreateNewRole
              trigger={<Button variant="secondary">Create New Role</Button>}
              permissions={key.workspace.permissions}
            />
            <CreateNewPermission
              trigger={<Button variant="secondary">Create New Permission</Button>}
            />
          </div>
        </div>

        <Chart
          apiId={props.params.apiId}
          key={JSON.stringify(key)}
          data={key}
          roles={key.workspace.roles.map((r) => ({
            ...r,
            active: key.roles.some((keyRole) => keyRole.roleId === r.id),
          }))}
          permissions={key.workspace.permissions.map((p) => ({
            ...p,
            active: transientPermissionIds.has(p.id),
          }))}
        />
      </div>
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
        getVerificationsPerInterval: getVerificationsHourly,
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
        getVerificationsPerInterval: getVerificationsDaily,
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
        getVerificationsPerInterval: getVerificationsDaily,
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
        getVerificationsPerInterval: getVerificationsDaily,
      };
    }
  }
}

const Metric: React.FC<{ label: string; value: React.ReactNode }> = ({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-between h-full px-4 py-2">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
    </div>
  );
};
