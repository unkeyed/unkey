import { AreaChart, StackedColumnChart } from "@/components/dashboard/charts";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import {
  getActiveKeys,
  getActiveKeysDaily,
  getActiveKeysHourly,
  getVerificationsDaily,
  getVerificationsHourly,
} from "@/lib/tinybird";
import { redirect } from "next/navigation";
import { type Interval, IntervalSelect } from "./select";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPage(props: {
  params: { apiId: string };
  searchParams: {
    interval?: Interval;
  };
}) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.apiId), isNull(table.deletedAt)),
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
    apiId: api.id,
    start,
    end,
  };
  const [
    keys,
    verifications,
    activeKeys,
    activeKeysTotal,
    _activeKeysInBillingCycle,
    verificationsInBillingCycle,
  ] = await Promise.all([
    db
      .select({ count: sql<number>`count(*)` })
      .from(schema.keys)
      .where(and(eq(schema.keys.keyAuthId, api.keyAuthId!), isNull(schema.keys.deletedAt)))
      .execute()
      .then((res) => res.at(0)?.count ?? 0),
    getVerificationsPerInterval(query),
    getActiveKeysPerInterval(query),
    getActiveKeys(query),
    getActiveKeys({
      workspaceId: api.workspaceId,
      apiId: api.id,
      start: billingCycleStart,
      end: billingCycleEnd,
    }).then((res) => res.data.at(0)),
    getVerificationsPerInterval({
      workspaceId: api.workspaceId,
      apiId: api.id,
      start: billingCycleStart,
      end: billingCycleEnd,
    }),
  ]);

  const successOverTime: { x: string; y: number }[] = [];
  const ratelimitedOverTime: { x: string; y: number }[] = [];
  const usageExceededOverTime: { x: string; y: number }[] = [];

  for (const d of verifications.data.sort((a, b) => a.time - b.time)) {
    const x = new Date(d.time).toISOString();
    successOverTime.push({ x, y: d.success });
    ratelimitedOverTime.push({ x, y: d.rateLimited });
    usageExceededOverTime.push({ x, y: d.usageExceeded });
  }

  const verificationsData = [
    ...successOverTime.map((d) => ({
      ...d,
      category: "Successful Verifications",
    })),
    ...ratelimitedOverTime.map((d) => ({ ...d, category: "Ratelimited" })),
    ...usageExceededOverTime.map((d) => ({ ...d, category: "Usage Exceeded" })),
  ];

  const activeKeysOverTime = activeKeys.data.map(({ time, keys }) => ({
    x: new Date(time).toISOString(),
    y: keys,
  }));

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="grid grid-cols-4 divide-x">
          <Metric label="Total Keys" value={formatNumber(keys)} />
          <Metric
            label={`Verifications in ${new Date().toLocaleString("en-US", {
              month: "long",
            })}`}
            value={formatNumber(
              verificationsInBillingCycle.data.reduce((sum, day) => sum + day.success, 0),
            )}
          />
          <Metric
            label={`Active Keys in ${new Date().toLocaleString("en-US", {
              month: "long",
            })}`}
            value={formatNumber(activeKeysTotal.data.at(0)?.keys ?? 0)}
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

      <Card>
        <CardHeader>
          <div className="grid grid-cols-3 divide-x">
            <Metric
              label="Successful Verifications"
              value={formatNumber(verifications.data.reduce((sum, day) => sum + day.success, 0))}
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
          </div>
        </CardHeader>
        <CardContent>
          <StackedColumnChart
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

      <Separator className="my-8" />
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-semibold leading-none tracking-tight">Active Keys</h2>

        <div>
          <IntervalSelect defaultSelected={interval} />
        </div>
      </div>
      <Card>
        <CardHeader>
          <div className="grid grid-cols-4 divide-x">
            <Metric
              label="Total Active Keys"
              value={formatNumber(activeKeysTotal.data.at(0)?.keys ?? 0)}
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
        getActiveKeysPerInterval: getActiveKeysHourly,
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
        getActiveKeysPerInterval: getActiveKeysDaily,
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
        getActiveKeysPerInterval: getActiveKeysDaily,
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
        getActiveKeysPerInterval: getActiveKeysDaily,
      };
    }
  }
}

const Metric: React.FC<{ label: string; value: string }> = ({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-center py-2 px-4">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
    </div>
  );
};
