import { AreaChart, StackedColumnChart } from "@/components/dashboard/charts";
import {
  Card,
  CardContent,
  CardHeader,
} from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { getActiveKeys, getActiveKeysDaily, getActiveKeysHourly, getTotalActiveKeys, getVerificationsDaily, getVerificationsHourly, getVerificationsMonthly, getVerificationsWeekly } from "@/lib/tinybird";
import { fillRange } from "@/lib/utils";
import { sql } from "drizzle-orm";
import { redirect } from "next/navigation";
import { type Interval, IntervalSelect } from "./select";
import { Separator } from "@/components/ui/separator";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPage(props: {
  params: { apiId: string }, searchParams: {
    interval?: Interval
  }
}) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return redirect("/new");
  }


  const interval = props.searchParams.interval ?? "24h";

  const keysP = db
    .select({ count: sql<number>`count(*)` })
    .from(schema.keys)
    .where(eq(schema.keys.keyAuthId, api.keyAuthId!))
    .execute()
    .then((res) => res.at(0)?.count ?? 0);



  const { getVerificationsPerInterval, getActiveKeysPerInterval, start, end, granularity } = prepareInterval(interval)
  const query = {
    workspaceId: api.workspaceId,
    apiId: api.id,
    start,
    end,
  }
  console.log({ query })
  const [usage, activeKeys, activeKeysTotal] = await Promise.all([
    getVerificationsPerInterval(query),
    getActiveKeysPerInterval(query),
    getActiveKeys(query)
  ])

  console.log({ activeKeys })


  const keys = await keysP;

  const successOverTime = fillRange(
    usage.data.map(({ time, success }) => ({ value: success, time })),
    start,
    end,
    granularity,
  ).map(({ value, time }) => ({
    x: new Date(time).toISOString(),
    y: value,
  }));

  const ratelimitedOverTime = fillRange(
    usage.data.map(({ time, rateLimited }) => ({ value: rateLimited, time })),
    start,
    end,
    granularity,
  ).map(({ value, time }) => ({
    x: new Date(time).toISOString(),
    y: value,
  }));

  const usageExceededOverTime = fillRange(
    usage.data.map(({ time, usageExceeded }) => ({ value: usageExceeded, time })),
    start,
    end,
    granularity,
  ).map(({ value, time }) => ({
    x: new Date(time).toISOString(),
    y: value,
  }));

  const verificationsData = [
    ...successOverTime.map((d) => ({
      ...d,
      category: "Successful Verifications",
    })),
    ...ratelimitedOverTime.map((d) => ({ ...d, category: "Ratelimited" })),
    ...usageExceededOverTime.map((d) => ({ ...d, category: "Usage Exceeded" })),
  ];


  const activeKeysOverTime = fillRange(
    activeKeys.data.map(({ time, keys }) => ({ value: keys, time })),
    start,
    end,
    granularity,
  ).map(({ value, time }) => ({
    x: new Date(time).toISOString(),
    y: value,
  }));

  return (
    <div className="flex flex-col gap-4">
      <Card >
        <CardContent className="grid grid-cols-3 divide-x">
          <Metric label="Total Keys" value={formatNumber(keys)} />
          {/* <Metric label="Ratelimited" value={formatNumber(usage.data.reduce((sum, day) => sum + day.rateLimited, 0))} />
            <Metric label="Usage Exceeded" value={formatNumber(usage.data.reduce((sum, day) => sum + day.usageExceeded, 0))} /> */}

        </CardContent>

      </Card>
      <Separator className="my-8" />

      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-semibold leading-none tracking-tight">Verifications</h2>

        <div>
          <IntervalSelect defaultSelected={interval} />
        </div>

      </div>


      <Card >
        <CardHeader>
          <div className="grid grid-cols-3 divide-x">
            <Metric label="Successful Verifications" value={formatNumber(usage.data.reduce((sum, day) => sum + day.success, 0))} />
            <Metric label="Ratelimited" value={formatNumber(usage.data.reduce((sum, day) => sum + day.rateLimited, 0))} />
            <Metric label="Usage Exceeded" value={formatNumber(usage.data.reduce((sum, day) => sum + day.usageExceeded, 0))} />
          </div>
        </CardHeader>
        <CardContent>
          <StackedColumnChart data={verificationsData} timeGranularity={granularity >= 1000 * 60 * 60 * 24 * 30 ? "month" : granularity >= 1000 * 60 * 60 * 24 ? "day" : "hour"} />
        </CardContent>
      </Card>

      <Separator className="my-8" />
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-semibold leading-none tracking-tight">Active Keys</h2>

        <div>
          <IntervalSelect defaultSelected={interval} />
        </div>



      </div>
      <Card >
        <CardHeader><div className="grid grid-cols-4 divide-x">
          <Metric label="Total Active Keys" value={formatNumber(activeKeysTotal.data.at(0)?.keys ?? 0)} />
        </div>
        </CardHeader>
        <CardContent>
          <AreaChart data={activeKeysOverTime} tooltipLabel="Active Keys" timeGranularity={granularity >= 1000 * 60 * 60 * 24 * 30 ? "month" : granularity >= 1000 * 60 * 60 * 24 ? "day" : "hour"} />
        </CardContent>
      </Card>
    </div >

  );
}


function prepareInterval(interval: Interval) {
  const now = new Date()

  switch (interval) {
    case "24h": {
      const end = now.setUTCHours(now.getUTCHours() + 1, 0, 0, 0)
      const intervalMs = 1000 * 60 * 60 * 24
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60,
        getVerificationsPerInterval: getVerificationsHourly,
        getActiveKeysPerInterval: getActiveKeysHourly,
      }
    }
    case "7d": {
      now.setUTCDate(now.getUTCDate() + 1)
      const end = now.setUTCHours(0, 0, 0, 0)
      const intervalMs = 1000 * 60 * 60 * 24 * 7
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60 * 24,
        getVerificationsPerInterval: getVerificationsDaily,
        getActiveKeysPerInterval: getActiveKeysDaily,

      }
    }
    case "30d": {
      now.setUTCDate(now.getUTCDate() + 1)
      const end = now.setUTCHours(0, 0, 0, 0)
      const intervalMs = 1000 * 60 * 60 * 24 * 30
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60 * 24,
        getVerificationsPerInterval: getVerificationsDaily,
        getActiveKeysPerInterval: getActiveKeysDaily,

      }
    }
    case "90d": {
      now.setUTCDate(now.getUTCDate() + 1)
      const end = now.setUTCHours(0, 0, 0, 0)
      const intervalMs = 1000 * 60 * 60 * 24 * 90
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60 * 24,
        getVerificationsPerInterval: getVerificationsDaily,
        getActiveKeysPerInterval: getActiveKeysDaily,

      }
    }

  }
}

const Metric: React.FC<{ label: string, value: string }> = ({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-center py-2 px-4">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
    </div>
  );
};
