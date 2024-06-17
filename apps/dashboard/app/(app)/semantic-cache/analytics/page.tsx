import { StackedColumnChart } from "@/components/dashboard/charts";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import {
  getAllSemanticCacheLogs,
  getSemanticCachesDaily,
  getSemanticCachesHourly,
} from "@/lib/tinybird";
import { redirect } from "next/navigation";
import { type Interval, IntervalSelect } from "../../apis/[apiId]/select";
import { getInterval } from "../logs/page";

type LogEntry = {
  hit: number;
  total: number;
  time: string;
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

  const gatewayId = workspace?.llmGateways[0]?.id;

  if (!gatewayId) {
    return redirect("/semantic-cache/new");
  }

  const { start, end, getSemanticCachesPerInterval } = prepareInterval(interval);

  const query = {
    start,
    end,
    gatewayId,
    workspaceId: workspace.id,
  };

  const { data: tokensData } = await getAllSemanticCacheLogs({
    limit: 1000,
    gatewayId,
    workspaceId: workspace.id,
    interval: getInterval(interval),
  });

  // const analyticsData = await _getSemanticCachesDaily();
  const { data: analyticsData } = await getSemanticCachesPerInterval(query);

  const tokens = tokensData.reduce((acc, log) => acc + log.tokens, 0);
  const timeSaved = tokensData.reduce(
    (acc, log) => acc + log.tokens / tokenCostMap[log.model].tps,
    0,
  );

  const transformLogs = (logs: LogEntry[]): TransformedEntry[] => {
    const transformedLogs: TransformedEntry[] = [];

    logs.forEach((log) => {
      const cacheHit: TransformedEntry = {
        x: log.time,
        y: log.hit,
        category: "Cache hit",
      };

      const cacheMiss: TransformedEntry = {
        x: log.time,
        y: log.total - log.hit,
        category: "Cache miss",
      };

      transformedLogs.push(cacheHit, cacheMiss);
    });

    return transformedLogs;
  };

  const transformedData = transformLogs(analyticsData);

  return (
    <div>
      <div className="flex py-4 text-gray-200">
        <Metric label="seconds saved" value={timeSaved.toFixed(5)} />
        <Metric label="tokens served from cache" value={tokens.toString()} />
      </div>
      <Separator />
      <div className="flex justify-end my-2">
        <IntervalSelect defaultSelected={"24h"} className="w-[200px]" />
      </div>
      <StackedColumnChart
        colors={["primary", "warn", "danger"]}
        data={transformedData}
        timeGranularity="hour"
      />
    </div>
  );
}

const Metric: React.FC<{ label: string; value: string }> = ({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-center px-4 py-2">
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
      <p className="text-sm text-content-subtle">{label}</p>
    </div>
  );
};
