import { AreaChart, StackedColumnChart } from "@/components/dashboard/charts";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageContent } from "@/components/page-content";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Metric } from "@/components/ui/metric";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { Empty } from "@unkey/ui";
import { redirect } from "next/navigation";
import { navigation } from "./constants";
import { Navigation } from "./navigation";
import { type Interval, IntervalSelect } from "./select";

export const dynamic = "force-dynamic";

export default async function ApiPage(props: {
  params: { apiId: string };
  searchParams: {
    interval?: Interval;
  };
}) {
  const tenantId = await getTenantId();

  const api = await db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.apiId), isNull(table.deletedAtM)),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return redirect("/new");
  }

  const interval = props.searchParams.interval ?? "7d";

  const t = new Date();
  t.setUTCDate(1);
  t.setUTCHours(0, 0, 0, 0);
  const billingCycleStart = t.getTime();
  const billingCycleEnd = t.setUTCMonth(t.getUTCMonth() + 1) - 1;
  const { getVerificationsPerInterval, getActiveKeysPerInterval, start, end, granularity } =
    prepareInterval(interval);
  const query = {
    workspaceId: api.workspaceId,
    keySpaceId: api.keyAuthId!,
    start,
    end,
  };
  const [keySpace, verifications, activeKeys, activeKeysTotal, verificationsInBillingCycle] =
    await Promise.all([
      db.query.keyAuth.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.id, api.keyAuthId!), isNull(table.deletedAtM)),
      }),
      getVerificationsPerInterval(query).then((res) => res.val!),
      getActiveKeysPerInterval(query).then((res) => res.val!),
      clickhouse.activeKeys
        .perMonth({
          workspaceId: api.workspaceId,
          keySpaceId: api.keyAuthId!,
          start: billingCycleStart,
          end: billingCycleEnd,
        })
        .then((res) => res.val!.at(0)),
      getVerificationsPerInterval({
        workspaceId: api.workspaceId,
        keySpaceId: api.keyAuthId!,
        start: billingCycleStart,
        end: billingCycleEnd,
      }).then((res) => res.val!),
    ]);

  const successOverTime: { x: string; y: number }[] = [];
  const ratelimitedOverTime: { x: string; y: number }[] = [];
  const usageExceededOverTime: { x: string; y: number }[] = [];
  const disabledOverTime: { x: string; y: number }[] = [];
  const insufficientPermissionsOverTime: { x: string; y: number }[] = [];
  const expiredOverTime: { x: string; y: number }[] = [];
  const forbiddenOverTime: { x: string; y: number }[] = [];
  const _emptyOverTime: { x: string; y: number }[] = [];

  for (const d of verifications.sort((a, b) => a.time - b.time)) {
    const x = new Date(d.time).toISOString();

    switch (d.outcome) {
      case "":
      case "VALID":
        successOverTime.push({ x, y: d.count });
        break;
      case "RATE_LIMITED":
        ratelimitedOverTime.push({ x, y: d.count });
        break;
      case "USAGE_EXCEEDED":
        usageExceededOverTime.push({ x, y: d.count });
        break;
      case "DISABLED":
        disabledOverTime.push({ x, y: d.count });
        break;
      case "INSUFFICIENT_PERMISSIONS":
        insufficientPermissionsOverTime.push({ x, y: d.count });
        break;
      case "EXPIRED":
        expiredOverTime.push({ x, y: d.count });
        break;
      case "FORBIDDEN":
        forbiddenOverTime.push({ x, y: d.count });
        break;
      default:
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
  ].sort((a, b) => new Date(a.x).getTime() - new Date(b.x).getTime());

  const activeKeysOverTime = activeKeys.map(({ time, keys }) => ({
    x: new Date(time).toISOString(),
    y: keys,
  }));

  return (
    <div>
      <Navigation api={api} />

      <PageContent>
        <SubMenu navigation={navigation(api.id, api.keyAuthId!)} segment="overview" />

        <div className="flex flex-col gap-4 mt-8">
          <Card>
            <CardContent className="grid grid-cols-3 divide-x">
              <Metric label="Total Keys" value={formatNumber(keySpace?.sizeApprox ?? 0)} />
              <Metric
                label={`Verifications in ${new Date().toLocaleString("en-US", {
                  month: "long",
                })}`}
                value={formatNumber(
                  verificationsInBillingCycle.reduce((sum, day) => sum + day.count, 0),
                )}
              />
              <Metric
                label={`Active Keys in ${new Date().toLocaleString("en-US", {
                  month: "long",
                })}`}
                value={formatNumber(activeKeysTotal?.keys ?? 0)}
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

          {verificationsData.some((d) => d.y > 0) ? (
            <Card>
              <CardHeader>
                <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 divide-x">
                  <Metric
                    label="Valid"
                    value={formatNumber(successOverTime.reduce((sum, day) => sum + day.y, 0))}
                  />
                  <Metric
                    label="Ratelimited"
                    value={formatNumber(ratelimitedOverTime.reduce((sum, day) => sum + day.y, 0))}
                  />
                  <Metric
                    label="Usage Exceeded"
                    value={formatNumber(usageExceededOverTime.reduce((sum, day) => sum + day.y, 0))}
                  />
                  <Metric
                    label="Disabled"
                    value={formatNumber(disabledOverTime.reduce((sum, day) => sum + day.y, 0))}
                  />
                  <Metric
                    label="Insufficient Permissions"
                    value={formatNumber(
                      insufficientPermissionsOverTime.reduce((sum, day) => sum + day.y, 0),
                    )}
                  />
                  <Metric
                    label="Expired"
                    value={formatNumber(expiredOverTime.reduce((sum, day) => sum + day.y, 0))}
                  />
                  <Metric
                    label="Forbidden"
                    value={formatNumber(forbiddenOverTime.reduce((sum, day) => sum + day.y, 0))}
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
            <Empty>
              <Empty.Icon />
              <Empty.Title>No usage</Empty.Title>
              <Empty.Description>Verify a key or change the range</Empty.Description>
            </Empty>
          )}

          <Separator className="my-8" />
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold leading-none tracking-tight">Active Keys</h2>

            <div>
              <IntervalSelect defaultSelected={interval} />
            </div>
          </div>
          {activeKeysOverTime.some((k) => k.y > 0) ? (
            <Card>
              <CardHeader>
                <div className="grid grid-cols-4 divide-x">
                  <Metric
                    label="Total Active Keys"
                    value={formatNumber(activeKeysTotal?.keys ?? 0)}
                  />
                </div>
              </CardHeader>
              <CardContent>
                <AreaChart
                  data={activeKeysOverTime}
                  tooltipLabel="Active Keys"
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
              <Empty.Title>No usage</Empty.Title>
              <Empty.Description>Verify a key or change the range</Empty.Description>
            </Empty>
          )}
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
        getActiveKeysPerInterval: clickhouse.activeKeys.perHour,
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
        getActiveKeysPerInterval: clickhouse.activeKeys.perDay,
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
        getActiveKeysPerInterval: clickhouse.activeKeys.perDay,
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
        getActiveKeysPerInterval: clickhouse.activeKeys.perDay,
      };
    }
  }
}
