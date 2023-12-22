import { StackedColumnChart } from "@/components/dashboard/charts";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import {
  getLastUsed,
  getLatestVerifications,
  getVerificationsDaily,
  getVerificationsHourly,
} from "@/lib/tinybird";
import { Minus } from "lucide-react";
import ms from "ms";
import { notFound } from "next/navigation";
import { type Interval, IntervalSelect } from "../../apis/[apiId]/select";
import { AccessTable } from "./table";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function KeyPage(props: {
  params: { keyId: string };
  searchParams: {
    interval?: Interval;
  };
}) {
  const tenantId = getTenantId();

  const key = await db.query.keys.findFirst({
    where: and(eq(schema.keys.id, props.params.keyId), isNull(schema.keys.deletedAt)),
    with: {
      workspace: true,
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

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="grid grid-cols-6 divide-x">
          <Metric
            label={key.expires && key.expires.getTime() < Date.now() ? "Expired" : "Expires"}
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
          <Metric
            label="Success"
            value={formatNumber(totalUsage.success)}
            tooltip="The total number of successful verifications for this key"
          />
          <Metric
            label="Ratelimited"
            value={formatNumber(totalUsage.rateLimited)}
            tooltip="The total number of ratelimited and therefore rejected verifications. These are not billed."
          />
          <Metric
            label="Usage Exceeded"
            value={formatNumber(totalUsage.usageExceeded)}
            tooltip="The total number of verifications that exceeded the limit and were rejected. These are not billed."
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
      <AccessTable verifications={latestVerifications.data} />
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

const Metric: React.FC<{
  label: React.ReactNode;
  value: React.ReactNode;
  tooltip?: React.ReactNode;
}> = ({ label, value, tooltip }) => {
  const component = (
    <div className="flex flex-col items-start justify-center py-2 px-4">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
    </div>
  );

  if (tooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>{component}</TooltipTrigger>
        <TooltipContent>
          <p className="text-sm text-content-subtle">{tooltip}</p>
        </TooltipContent>
      </Tooltip>
    );
  }
  return component;
};
// pnpm install  20.92s user 44.80s system 167% cpu 39.265 total
