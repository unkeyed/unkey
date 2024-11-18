import { StackedColumnChart } from "@/components/dashboard/charts";
import { CopyButton } from "@/components/dashboard/copy-button";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db, eq, schema, sql } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { BarChart } from "lucide-react";
import ms from "ms";
import { redirect } from "next/navigation";
import { parseAsArrayOf, parseAsString, parseAsStringEnum } from "nuqs/server";
import { Filters, type Interval } from "./filters";

export const dynamic = "force-dynamic";
export const runtime = "edge";

const intervalParser = parseAsStringEnum(["60m", "24h", "7d", "30d", "90d"]).withDefault("7d");

export default async function RatelimitNamespacePage(props: {
  params: { namespaceId: string };
  searchParams: {
    interval?: Interval;
    identifier?: string;
  };
}) {
  const tenantId = getTenantId();

  const namespace = await db.query.ratelimitNamespaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.namespaceId), isNull(table.deletedAt)),
    with: {
      workspace: {
        columns: {
          tenantId: true,
        },
      },
    },
  });
  if (!namespace || namespace.workspace.tenantId !== tenantId) {
    return redirect("/ratelimits");
  }

  const interval = intervalParser.withDefault("7d").parseServerSide(props.searchParams.interval);
  const selectedIdentifier = parseAsArrayOf(parseAsString)
    .withDefault([])
    .parseServerSide(props.searchParams.identifier);

  const t = new Date();
  t.setUTCDate(1);
  t.setUTCHours(0, 0, 0, 0);
  const billingCycleStart = t.getTime();
  const billingCycleEnd = t.setUTCMonth(t.getUTCMonth() + 1) - 1;

  const { getRatelimitsPerInterval, start, end, granularity } = prepareInterval(interval);
  const query = {
    workspaceId: namespace.workspaceId,
    namespaceId: namespace.id,
    start,
    end,
    identifier: selectedIdentifier.length > 0 ? selectedIdentifier : undefined,
  };
  const [customLimits, ratelimitEvents, ratelimitsInBillingCycle, lastUsed] = await Promise.all([
    db
      .select({ count: sql<number>`count(*)` })
      .from(schema.ratelimitOverrides)
      .where(eq(schema.ratelimitOverrides.namespaceId, namespace.id))
      .execute()
      .then((res) => res?.at(0)?.count ?? 0),
    getRatelimitsPerInterval(query).then((res) => res.val!),
    getRatelimitsPerInterval({
      workspaceId: namespace.workspaceId,
      namespaceId: namespace.id,
      start: billingCycleStart,
      end: billingCycleEnd,
    }).then((res) => res.val!),
    clickhouse.ratelimits
      .latest({ workspaceId: namespace.workspaceId, namespaceId: namespace.id })
      .then((res) => res.val?.at(0)?.time),
  ]);

  const passedOverTime: { x: string; y: number }[] = [];
  const ratelimitedOverTime: { x: string; y: number }[] = [];

  for (const d of ratelimitEvents.sort((a, b) => a.time - b.time)) {
    const x = new Date(d.time).toISOString();
    passedOverTime.push({ x, y: d.passed });
    ratelimitedOverTime.push({ x, y: d.total - d.passed });
  }

  // const dataOverTime = [
  //   ...ratelimitedOverTime.map((d) => ({ ...d, category: "Ratelimited" })),
  //   ...successOverTime.map((d) => ({
  //     ...d,
  //     category: "Successful",
  //   })),
  // ];
  const dataOverTime = ratelimitEvents.flatMap((d) => [
    {
      x: new Date(d.time).toISOString(),
      y: d.total - d.passed,
      category: "Ratelimited",
    },
    {
      x: new Date(d.time).toISOString(),
      y: d.passed,
      category: "Passed",
    },
  ]);

  // const activeKeysOverTime = activeKeys.data.map(({ time, keys }) => ({
  //   x: new Date(time).toISOString(),
  //   y: keys,
  // }));

  const snippet = `curl -XPOST 'https://api.unkey.dev/v1/ratelimits.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -d '{
      "namespace": "${namespace.name}",
      "identifier": "<USER_ID>",
      "limit": 10,
      "duration": 10000
  }'`;
  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="grid grid-cols-3 divide-x">
          <Metric label="Overriden limits" value={formatNumber(customLimits)} />
          <Metric
            label={`Successful ratelimits in ${new Date().toLocaleString("en-US", {
              month: "long",
            })}`}
            value={formatNumber(ratelimitsInBillingCycle.reduce((sum, day) => sum + day.passed, 0))}
          />
          <Metric
            label="Last used"
            value={lastUsed ? `${ms(Date.now() - lastUsed)} ago` : "never"}
          />

          {/* <Metric
            label={`Active Keys in ${new Date().toLocaleString("en-US", {
              month: "long",
            })}`}
            value={formatNumber(activeKeysTotal.data.at(0)?.keys ?? 0)}
          /> */}
        </CardContent>
      </Card>
      <Separator className="my-8" />

      <div className="flex items-center justify-between w-full">
        <div>
          <h2 className="text-2xl font-semibold leading-none tracking-tight whitespace-nowrap">
            Requests
          </h2>
        </div>

        <Filters identifier interval />
      </div>

      {dataOverTime.some((d) => d.y > 0) ? (
        <Card>
          <CardHeader>
            <div className="grid grid-cols-2 lg:grid-cols-4 lg:divide-x">
              <Metric
                label="Passed"
                value={formatNumber(ratelimitEvents.reduce((sum, day) => sum + day.passed, 0))}
              />
              <Metric
                label="Ratelimited"
                value={formatNumber(
                  ratelimitEvents.reduce((sum, day) => sum + (day.total - day.passed), 0),
                )}
              />
              <Metric
                label="Total"
                value={formatNumber(ratelimitEvents.reduce((sum, day) => sum + day.total, 0))}
              />
              <Metric
                label="Success Rate"
                value={`${formatNumber(
                  (ratelimitEvents.reduce((sum, day) => sum + day.passed, 0) /
                    ratelimitEvents.reduce((sum, day) => sum + day.total, 0)) *
                    100,
                )}%`}
              />
            </div>
          </CardHeader>
          <CardContent>
            <StackedColumnChart
              colors={["warn", "primary"]}
              data={dataOverTime}
              timeGranularity={
                granularity >= 1000 * 60 * 60 * 24 * 30
                  ? "month"
                  : granularity >= 1000 * 60 * 60 * 24
                    ? "day"
                    : granularity >= 1000 * 60 * 60
                      ? "hour"
                      : "minute"
              }
            />
          </CardContent>
        </Card>
      ) : (
        <EmptyPlaceholder>
          <EmptyPlaceholder.Icon>
            <BarChart />
          </EmptyPlaceholder.Icon>
          <EmptyPlaceholder.Title>No usage</EmptyPlaceholder.Title>
          <EmptyPlaceholder.Description>
            Ratelimit something or change the range
          </EmptyPlaceholder.Description>
          <Code className="flex items-start gap-8 p-4 my-8 text-xs text-left">
            {snippet}
            <CopyButton value={snippet} />
          </Code>
        </EmptyPlaceholder>
      )}
    </div>
  );
}

function prepareInterval(interval: Interval) {
  const now = new Date();

  switch (interval) {
    case "60m": {
      const end = now.setUTCMinutes(now.getUTCMinutes() + 1, 0, 0);
      const intervalMs = 1000 * 60 * 60;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60,
        getRatelimitsPerInterval: clickhouse.ratelimits.perMinute,
      };
    }
    case "24h": {
      const end = now.setUTCHours(now.getUTCHours() + 1, 0, 0, 0);
      const intervalMs = 1000 * 60 * 60 * 24;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60,
        getRatelimitsPerInterval: clickhouse.ratelimits.perHour,
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
        getRatelimitsPerInterval: clickhouse.ratelimits.perDay,
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
        getRatelimitsPerInterval: clickhouse.ratelimits.perDay,
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
        getRatelimitsPerInterval: clickhouse.ratelimits.perDay,
      };
    }
  }
}

const Metric: React.FC<{ label: string; value: string }> = ({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-center px-4 py-2">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
    </div>
  );
};
