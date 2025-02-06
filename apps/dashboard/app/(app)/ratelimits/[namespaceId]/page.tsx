import { StackedColumnChart } from "@/components/dashboard/charts";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { Metric } from "@/components/ui/metric";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { Gauge } from "@unkey/icons";
import { Empty } from "@unkey/ui";
import ms from "ms";
import { redirect } from "next/navigation";
import { parseAsArrayOf, parseAsString, parseAsStringEnum } from "nuqs/server";
import { navigation } from "./constants";
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
  const timeseriesMethod = getRatelimitsPerInterval.replace("clickhouse.ratelimits.", "") as
    | "perMinute"
    | "perHour"
    | "perDay"
    | "perMonth";
  const query = {
    workspaceId: namespace.workspaceId,
    namespaceId: namespace.id,
    startTime: start,
    endTime: end,
    identifiers:
      selectedIdentifier.length > 0
        ? selectedIdentifier.map((id) => ({
            operator: "is" as const,
            value: id,
          }))
        : null,
  };

  const [customLimits, ratelimitEvents, ratelimitsInBillingCycle, lastUsed] = await Promise.all([
    db
      .select({ count: sql<number>`count(*)` })
      .from(schema.ratelimitOverrides)
      .where(
        and(
          eq(schema.ratelimitOverrides.namespaceId, namespace.id),
          isNull(schema.ratelimitOverrides.deletedAt),
        ),
      )
      .execute()
      .then((res) => res?.at(0)?.count ?? 0),
    clickhouse.ratelimits.timeseries[timeseriesMethod](query),
    clickhouse.ratelimits.timeseries[timeseriesMethod]({
      ...query,
      startTime: billingCycleStart,
      endTime: billingCycleEnd,
    }),
    clickhouse.ratelimits
      .latest({
        workspaceId: namespace.workspaceId,
        namespaceId: namespace.id,
        limit: 1,
      })
      .then((res) => res.val?.at(0)?.time),
  ]);

  const dataOverTime = (ratelimitEvents.val ?? []).flatMap((event) => [
    {
      x: new Date(event.x).toISOString(),
      y: event.y.total - event.y.passed,
      category: "Ratelimited",
    },
    {
      x: new Date(event.x).toISOString(),
      y: event.y.passed,
      category: "Passed",
    },
  ]);

  const totalPassed = (ratelimitEvents.val ?? []).reduce((sum, event) => sum + event.y.passed, 0);
  const totalRatelimited = (ratelimitEvents.val ?? []).reduce(
    (sum, event) => sum + (event.y.total - event.y.passed),
    0,
  );
  const totalRequests = totalPassed + totalRatelimited;
  const successRate = totalRequests > 0 ? (totalPassed / totalRequests) * 100 : 0;

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
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href="/ratelimits">Ratelimits</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/ratelimits/${props.params.namespaceId}`}
            isIdentifier
            active
          >
            {namespace.name.length > 0 ? namespace.name : "<Empty>"}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="namespaceId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {props.params.namespaceId}
            <CopyButton value={props.params.namespaceId} />
          </Badge>
        </Navbar.Actions>
      </Navbar>
      <PageContent>
        <SubMenu navigation={navigation(props.params.namespaceId)} segment="overview" />
        <div className="flex flex-col gap-4 mt-8">
          <Card>
            <CardContent className="grid grid-cols-1 divide-y md:grid-cols-3 md:divide-y-0 md:divide-x">
              <Metric label="Overriden limits" value={formatNumber(customLimits)} />
              <Metric
                label={`Successful ratelimits in ${new Date().toLocaleString("en-US", {
                  month: "long",
                })}`}
                value={formatNumber(
                  (ratelimitsInBillingCycle.val ?? []).reduce((sum, day) => sum + day.y.passed, 0),
                )}
              />
              <Metric
                label="Last used"
                value={lastUsed ? `${ms(Date.now() - lastUsed)} ago` : "never"}
              />
            </CardContent>
          </Card>
          <Separator className="my-8" />

          <div className="flex items-center justify-between w-full">
            <div>
              <h2 className="text-2xl font-semibold leading-none tracking-tight whitespace-nowrap hidden sm:block">
                Requests
              </h2>
            </div>

            <Filters identifier interval />
          </div>

          {dataOverTime.some((d) => d.y > 0) ? (
            <Card>
              <CardHeader>
                <div className="grid grid-cols-2 lg:grid-cols-4 lg:divide-x">
                  <Metric label="Passed" value={formatNumber(totalPassed)} />
                  <Metric label="Ratelimited" value={formatNumber(totalRatelimited)} />
                  <Metric label="Total" value={formatNumber(totalRequests)} />
                  <Metric label="Success Rate" value={`${formatNumber(successRate)}%`} />
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
            <Empty>
              <Empty.Icon />
              <Empty.Title>No usage</Empty.Title>
              <Empty.Description>Ratelimit something or change the range</Empty.Description>
              <Code className="flex items-start gap-0 sm:gap-8 p-4 my-8 text-xs sm:text-xxs text-start overflow-x-auto max-w-full">
                {snippet}
                <CopyButton value={snippet} />
              </Code>
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
    case "60m": {
      const end = now.setUTCMinutes(now.getUTCMinutes() + 1, 0, 0);
      const intervalMs = 1000 * 60 * 60;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60,
        getRatelimitsPerInterval: "perMinute",
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
        getRatelimitsPerInterval: "perHour",
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
        getRatelimitsPerInterval: "perDay",
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
        getRatelimitsPerInterval: "perDay",
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
        getRatelimitsPerInterval: "perDay",
      };
    }
  }
}
