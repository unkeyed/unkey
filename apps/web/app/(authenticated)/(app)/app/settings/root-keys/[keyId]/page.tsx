import { StackedColumnChart } from "@/components/dashboard/charts";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import {
  getDailyVerifications,
  getLastUsed,
  getLatestVerifications,
  getTotalVerificationsForKey,
} from "@/lib/tinybird";
import { fillRange } from "@/lib/utils";
import { Check, Info, Minus } from "lucide-react";
import ms from "ms";
import { notFound } from "next/navigation";
export const revalidate = 0;

export default async function Page(props: { params: { keyId: string } }) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return notFound();
  }
  if (workspace.tenantId !== tenantId) {
    return notFound();
  }

  const key = await db.query.keys.findFirst({
    where: eq(schema.keys.forWorkspaceId, workspace.id) && eq(schema.keys.id, props.params.keyId),
  });
  if (!key) {
    return notFound();
  }

  const [usage, totalVerifications, latestVerifications, lastUsed] = await Promise.all([
    getDailyVerifications({
      workspaceId: env().UNKEY_WORKSPACE_ID,
      apiId: env().UNKEY_API_ID,
      keyId: key.id,
    }),
    getTotalVerificationsForKey({ keyId: key.id }).then((res) => res.data.at(0)?.totalUsage ?? 0),
    getLatestVerifications({ keyId: key.id }),
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
        <CardContent className="grid grid-cols-2 gap-px mx-auto xl:divide-x md:grid-cols-3 xl:grid-cols-6 ">
          <Stat label="Usage 30 days" value={fmt(usage30Days)} />
          <Stat
            label="Expires"
            value={key.expires ? ms(key.expires.getTime() - Date.now()) : <Minus />}
          />
          <Stat
            label="Remaining"
            value={
              typeof key.remainingRequests === "number" ? fmt(key.remainingRequests) : <Minus />
            }
          />
          <Stat
            label="LastUsed"
            value={lastUsed ? `${ms(Date.now() - lastUsed)} ago` : <Minus />}
          />
          <Stat label="Total Uses" value={fmt(totalVerifications)} />
          <Stat
            label={
              <Tooltip>
                <TooltipTrigger className="flex items-center gap-1">
                  Key ID <Info className="w-4 h-4" />
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

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>Resource</TableHead>
            <TableHead>User Agent</TableHead>
            <TableHead>IP Address</TableHead>
            <TableHead>Region</TableHead>
            <TableHead>Valid</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {latestVerifications.data?.map((verification, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
            <TableRow key={i}>
              <TableCell className="flex flex-col">
                <span className="text-content">{new Date(verification.time).toDateString()}</span>
                <span className="text-xs text-content-subtle">
                  {new Date(verification.time).toTimeString().split("(").at(0)}
                </span>
              </TableCell>
              <TableCell>{verification.requestedResource}</TableCell>
              <TableCell className="max-w-sm truncate">{verification.userAgent}</TableCell>
              <TableCell>{verification.ipAddress}</TableCell>
              <TableCell>{verification.region}</TableCell>
              <TableCell>
                {verification.usageExceeded ? (
                  <Badge>Usage Exceede</Badge>
                ) : verification.ratelimited ? (
                  <Badge>Ratelimited</Badge>
                ) : (
                  <Check />
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

const Stat: React.FC<{ label: React.ReactNode; value: React.ReactNode }> = ({ label, value }) => (
  <div className="flex flex-wrap items-baseline justify-between px-4 py-2 gap-x-4 gap-y-2 sm:px-6 xl:px-8">
    <dt className="text-sm font-medium leading-6 text-content-subtle">{label}</dt>

    <dd className="flex-none w-full text-3xl font-medium leading-10 tracking-tight text-content">
      {value}
    </dd>
  </div>
);
