import { StackedColumnChart } from "@/components/dashboard/charts";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema } from "@/lib/db";
import {
  getDailyVerifications,
  getLastUsed,
  getLatestVerifications,
  getTotalVerificationsForKey,
} from "@/lib/tinybird";
import { fillRange } from "@/lib/utils";
import { Info, Minus } from "lucide-react";
import ms from "ms";
import { notFound } from "next/navigation";
import { AccessTable } from "./table";

export default async function KeyPage(props: { params: { keyId: string } }) {
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
    where: eq(schema.apis.keyAuthId, key.keyAuthId),
  });
  if (!api) {
    return notFound();
  }

  const [usage, totalUsage, latestVerifications, lastUsed] = await Promise.all([
    getDailyVerifications({
      workspaceId: key.workspaceId,
      apiId: api.id,
      keyId: key.id,
    }),
    getTotalVerificationsForKey({ keyId: key.id }).then((res) => res.data.at(0)?.totalUsage ?? 0),
    getLatestVerifications({
      workspaceId: key.workspaceId,
      apiId: api.id,
      keyId: key.id,
    }),
    getLastUsed({ keyId: key.id }).then((res) => res.data.at(0)?.lastUsed ?? 0),
  ]);

  const end = new Date().setUTCHours(0, 0, 0, 0);
  const start = end - 30 * 24 * 60 * 60 * 1000;

  const usageOverTime = [
    ...fillRange(
      usage.data.map(({ time, success }) => ({ value: success, time })),
      start,
      end,
    ).map((x) => ({ ...x, category: "Success" })),
    ...fillRange(
      usage.data.map(({ time, rateLimited }) => ({
        value: rateLimited,
        time,
      })),
      start,
      end,
    ).map((x) => ({ ...x, category: "Rate Limited" })),
    ...fillRange(
      usage.data.map(({ time, usageExceeded }) => ({
        value: usageExceeded,
        time,
      })),
      start,
      end,
    ).map((x) => ({ ...x, category: "Usage Exceeded" })),
  ].map(({ value, time, category }) => ({
    category,
    x: new Date(time).toUTCString(),
    y: value,
  }));

  const fmt = new Intl.NumberFormat("en-US", { notation: "compact" }).format;
  const usage30Days = usage.data.reduce(
    (acc, { success, rateLimited, usageExceeded }) => acc + success + rateLimited + usageExceeded,
    0,
  );

  return (
    <div className="flex flex-col gap-8">
      <Card>
        <CardContent className="mx-auto grid grid-cols-2 gap-px md:grid-cols-3 xl:grid-cols-6 xl:divide-x ">
          <Stat label="Usage 30 days" value={fmt(usage30Days)} />
          <Stat
            label={key.expires && key.expires.getTime() < Date.now() ? "Expired" : "Expires"}
            value={key.expires ? ms(key.expires.getTime() - Date.now()) : <Minus />}
          />
          <Stat
            label="Remaining"
            value={
              typeof key.remainingRequests === "number" ? fmt(key.remainingRequests) : <Minus />
            }
          />
          <Stat
            label="Last Used"
            value={lastUsed ? `${ms(Date.now() - lastUsed)} ago` : <Minus />}
          />
          <Stat label="Total Uses" value={fmt(totalUsage)} />
          <Stat
            label={
              <Tooltip>
                <TooltipTrigger className="flex items-center gap-1">
                  Key ID <Info className="h-4 w-4" />
                </TooltipTrigger>
                <TooltipContent>
                  This is not the secret key, but just a unique identifier used for interacting with
                  our API.
                </TooltipContent>
              </Tooltip>
            }
            value={
              <Badge
                key="keyId"
                variant="secondary"
                className="flex justify-between font-mono font-medium "
              >
                <span className="truncate">{key.id}</span>
                <CopyButton value={key.id} className="ml-2" />
              </Badge>
            }
          />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Usage in the last 30 days</CardTitle>
          <CardDescription>See when this key was verified</CardDescription>
        </CardHeader>
        <CardContent className="h-64">
          <StackedColumnChart data={usageOverTime} />
        </CardContent>
      </Card>

      <AccessTable verifications={latestVerifications.data} />
    </div>
  );
}

const Stat: React.FC<{ label: React.ReactNode; value: React.ReactNode }> = ({ label, value }) => (
  <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-2 px-4 py-2 sm:px-6 xl:px-8">
    <dt className="text-content-subtle text-sm font-medium leading-6">{label}</dt>

    <dd className="text-content w-full flex-none text-3xl font-medium leading-10 tracking-tight">
      {value}
    </dd>
  </div>
);
