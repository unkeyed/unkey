import { StackedBarChart, StackedColumnChart } from "@/components/dashboard/charts";
import { CopyButton } from "@/components/dashboard/copy-button";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema, sql } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import {
  getRatelimitIdentifiersDaily,
  getRatelimitIdentifiersHourly,
  getRatelimitIdentifiersMinutely,
  getRatelimitIdentifiersMonthly,
  getRatelimitLastUsed,
  getRatelimitsDaily,
  getRatelimitsHourly,
  getRatelimitsMinutely,
} from "@/lib/tinybird";
import { BarChart } from "lucide-react";
import ms from "ms";
import { notFound } from "next/navigation";
import { parseAsArrayOf, parseAsString, parseAsStringEnum } from "nuqs";
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
    return notFound();
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

  const { getRatelimitsPerInterval, getIdentifiers, start, end, granularity } =
    prepareInterval(interval);
  const query = {
    workspaceId: namespace.workspaceId,
    namespaceId: namespace.id,
    start,
    end,
    identifier: selectedIdentifier.length > 0 ? selectedIdentifier : undefined,
  };
  const [customLimits, ratelimitEvents, identifiers, ratelimitsInBillingCycle, lastUsed] =
    await Promise.all([
      db
        .select({ count: sql<number>`count(*)` })
        .from(schema.ratelimitOverrides)
        .where(eq(schema.ratelimitOverrides.namespaceId, namespace.id))
        .execute()
        .then((res) => res.at(0)?.count ?? 0),
      getRatelimitsPerInterval(query),
      getIdentifiers(query),
      getRatelimitsPerInterval({
        workspaceId: namespace.workspaceId,
        namespaceId: namespace.id,
        start: billingCycleStart,
        end: billingCycleEnd,
      }),
      getRatelimitLastUsed({ workspaceId: namespace.workspaceId, namespaceId: namespace.id }).then(
        (res) => res.data.at(0)?.lastUsed,
      ),
    ]);

  const successOverTime: { x: string; y: number }[] = [];
  const ratelimitedOverTime: { x: string; y: number }[] = [];

  for (const d of ratelimitEvents.data.sort((a, b) => a.time - b.time)) {
    const x = new Date(d.time).toISOString();
    successOverTime.push({ x, y: d.success });
    ratelimitedOverTime.push({ x, y: d.total - d.success });
  }

  // const dataOverTime = [
  //   ...ratelimitedOverTime.map((d) => ({ ...d, category: "Ratelimited" })),
  //   ...successOverTime.map((d) => ({
  //     ...d,
  //     category: "Successful",
  //   })),
  // ];
  const dataOverTime = ratelimitEvents.data.flatMap((d) => [
    {
      x: new Date(d.time).toISOString(),
      y: d.total - d.success,
      category: "Ratelimited",
    },
    {
      x: new Date(d.time).toISOString(),
      y: d.success,
      category: "Success",
    },
  ]);

  // const activeKeysOverTime = activeKeys.data.map(({ time, keys }) => ({
  //   x: new Date(time).toISOString(),
  //   y: keys,
  // }));

  const snippet = `curl -XPOST 'https://api.unkey.dev/v1/ratelimits.limit' \\
  -h 'Content-Type: application/json' \\
  -h 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
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
            value={formatNumber(
              ratelimitsInBillingCycle.data.reduce((sum, day) => sum + day.success, 0),
            )}
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
            <div className="grid grid-cols-4 lg:grid-cols-4 lg:divide-x">
              <Metric
                label="Successful"
                value={formatNumber(
                  ratelimitEvents.data.reduce((sum, day) => sum + day.success, 0),
                )}
              />
              <Metric
                label="Ratelimited"
                value={formatNumber(
                  ratelimitEvents.data.reduce((sum, day) => sum + (day.total - day.success), 0),
                )}
              />
              <Metric
                label="Total"
                value={formatNumber(ratelimitEvents.data.reduce((sum, day) => sum + day.total, 0))}
              />
              <Metric
                label="Success Rate"
                value={`${formatNumber(
                  (ratelimitEvents.data.reduce((sum, day) => sum + day.success, 0) /
                    ratelimitEvents.data.reduce((sum, day) => sum + day.total, 0)) *
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
            <Code className="flex items-start gap-8 p-4 my-8 text-xs text-left">
              {snippet}
              <CopyButton value={snippet} />
            </Code>
          </EmptyPlaceholder.Description>
        </EmptyPlaceholder>
      )}

      <Separator className="my-8" />
      <div className="flex items-center justify-between w-full">
        <h2 className="text-2xl font-semibold leading-none tracking-tight whitespace-nowrap">
          Top identifiers
        </h2>

        <Filters interval />
      </div>
      {identifiers.data.length > 0 ? (
        <Card>
          <CardContent>
            <StackedBarChart
              colors={["primary", "warn"]}
              data={identifiers.data.slice(0, 10).flatMap(({ identifier, success, total }) => [
                {
                  x: success,
                  y: identifier,
                  category: "Success",
                },
                {
                  x: total - success,
                  y: identifier,
                  category: "Ratelimited",
                },
              ])}
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
            <Code className="flex items-start gap-8 p-4 my-8 text-xs text-left">
              {snippet}
              <CopyButton value={snippet} />
            </Code>
          </EmptyPlaceholder.Description>
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
        getRatelimitsPerInterval: getRatelimitsMinutely,
        getIdentifiers: getRatelimitIdentifiersMinutely,
        // getActiveKeysPerInterval: getActiveKeysHourly,
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
        getRatelimitsPerInterval: getRatelimitsHourly,
        getIdentifiers: getRatelimitIdentifiersHourly,

        // getActiveKeysPerInterval: getActiveKeysHourly,
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
        getRatelimitsPerInterval: getRatelimitsDaily,
        getIdentifiers: getRatelimitIdentifiersDaily,

        // getActiveKeysPerInterval: getActiveKeysDaily,
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
        getRatelimitsPerInterval: getRatelimitsDaily,
        getIdentifiers: getRatelimitIdentifiersDaily,

        // getActiveKeysPerInterval: getActiveKeysDaily,
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
        getRatelimitsPerInterval: getRatelimitsDaily,
        getIdentifiers: getRatelimitIdentifiersMonthly,

        // getActiveKeysPerInterval: getActiveKeysDaily,
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
