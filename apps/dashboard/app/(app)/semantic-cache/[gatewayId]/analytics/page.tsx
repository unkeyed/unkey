import { StackedColumnChart } from "@/components/dashboard/charts";
import { CopyButton } from "@/components/dashboard/copy-button";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Card, CardContent } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import {
  getAllSemanticCacheLogs,
  getSemanticCachesDaily,
  getSemanticCachesHourly,
} from "@/lib/tinybird";
import { BarChart } from "lucide-react";
import ms from "ms";
import { redirect } from "next/navigation";
import { type Interval, IntervalSelect } from "../../../apis/[apiId]/select";

type LogEntry = {
  hit: number;
  total: number;
  time: number;
};

type TransformedEntry = {
  x: string;
  y: number;
  category: string;
};

const tokenCostMap = {
  "gpt-4o": { cost: 15 / 1_000_000, tps: 63.32 },
  "gpt-4-turbo": { cost: 10 / 1_000_000, tps: 35.68 },
  "gpt-4": { cost: 30 / 1_000_000, tps: 35.68 },
  "gpt-3.5-turbo-0125": { cost: 0.5 / 1_000_000, tps: 67.84 },
  "gpt-3.5-turbo": { cost: 0.5 / 1_000_000, tps: 67.84 },
} as { [key: string]: { cost: number; tps: number } };

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
        getSemanticCachesPerInterval: getSemanticCachesHourly,
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
        getSemanticCachesPerInterval: getSemanticCachesDaily,
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
        getSemanticCachesPerInterval: getSemanticCachesDaily,
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
        getSemanticCachesPerInterval: getSemanticCachesDaily,
      };
    }
  }
}

export default async function SemanticCacheAnalyticsPage(props: {
  searchParams: { interval?: Interval };
}) {
  const interval = props.searchParams.interval ?? "24h";

  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      llmGateways: {
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  const gateway = workspace.llmGateways.at(0);

  if (!gateway) {
    return redirect("/semantic-cache/new");
  }

  const { start, end, granularity, getSemanticCachesPerInterval } = prepareInterval(interval);

  const query = {
    start,
    end,
    gatewayId: gateway.id,
    workspaceId: workspace.id,
  };

  const { data: analyticsData } = await getSemanticCachesPerInterval(query);

  const cachedTokens = analyticsData.reduce((acc, log) => acc + log.cachedTokens, 0);
  // const totalTokens = analyticsData.reduce((acc, log) => acc + log.sumTokens, 0);
  const dollarSaved = analyticsData.reduce((acc, log) => {
    const cost = tokenCostMap[log.model || "gpt-4"];
    if (cost) {
      return acc + log.cachedTokens * cost.cost;
    }
    return acc + log.cachedTokens / tokenCostMap["gpt-4"].cost;
  }, 0);
  const millisecondsSaved =
    1000 *
    analyticsData.reduce((acc, log) => {
      const cost = tokenCostMap[log.model || "gpt-4"];
      if (cost) {
        return acc + log.sumTokens / cost.tps;
      }
      return acc + log.sumTokens / tokenCostMap["gpt-4"].tps;
    }, 0);

  const transformLogs = (logs: LogEntry[]): TransformedEntry[] => {
    const transformedLogs: TransformedEntry[] = [];

    logs.forEach((log) => {
      const cacheHit: TransformedEntry = {
        x: new Date(log.time).toISOString(),
        y: log.hit,
        category: "Cache hit",
      };

      const cacheMiss: TransformedEntry = {
        x: new Date(log.time).toISOString(),
        y: log.total - log.hit,
        category: "Cache miss",
      };

      transformedLogs.push(cacheHit, cacheMiss);
    });

    return transformedLogs;
  };

  const transformedData = transformLogs(analyticsData);

  const snippet = `const openai = new OpenAI({
    apiKey: process.env.OPENAI_API_KEY,
    baseURL: "https://${gateway.name}.llm.unkey.io",
});`;

  return (
    <div className="space-y-4 ">
      <Card>
        <CardContent className="grid grid-cols-3 divide-x">
          <Metric label="Time saved" value={ms(Math.floor(millisecondsSaved))} />
          <Metric
            label="Tokens served from cache"
            value={Intl.NumberFormat(undefined, { notation: "compact" }).format(cachedTokens)}
          />
          <Metric
            label="Money saved"
            value={`$${Intl.NumberFormat(undefined, { currency: "USD" }).format(dollarSaved)}`}
          />
        </CardContent>
      </Card>
      <Separator />
      <div className="flex justify-end my-2">
        <IntervalSelect defaultSelected={"24h"} className="w-[200px]" />
      </div>
      {transformedData.some((d) => d.y) ? (
        <StackedColumnChart
          colors={["primary", "warn", "danger"]}
          data={transformedData}
          timeGranularity={
            granularity >= 1000 * 60 * 60 * 24 * 30
              ? "month"
              : granularity >= 1000 * 60 * 60 * 24
                ? "day"
                : "hour"
          }
        />
      ) : (
        <EmptyPlaceholder>
          <EmptyPlaceholder.Icon>
            <BarChart />
          </EmptyPlaceholder.Icon>
          <EmptyPlaceholder.Title>No usage</EmptyPlaceholder.Title>
          <EmptyPlaceholder.Description>
            Use the snippet below to start using the semantic cache.
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

const Metric: React.FC<{
  label: React.ReactNode;
  value: React.ReactNode;
}> = ({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-center px-4 py-2">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
    </div>
  );
};
