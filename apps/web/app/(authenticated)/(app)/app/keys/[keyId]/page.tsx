import { ColumnChart } from "@/components/dashboard/charts";
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
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { getDailyUsage, getLatestVerifications } from "@/lib/tinybird";
import { fillRange } from "@/lib/utils";
import { Minus } from "lucide-react";
import ms from "ms";
import { notFound } from "next/navigation";
export const revalidate = 0;

export default async function KeyPage(props: { params: { keyId: string } }) {
  const tenantId = getTenantId();

  const key = await db.query.keys.findFirst({
    where: eq(schema.keys.id, props.params.keyId),
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
  console.log(key.workspaceId, api.id, key.id);
  const usage = await getDailyUsage({
    workspaceId: key.workspaceId,
    apiId: api.id,
    keyId: key.id,
  });

  const end = new Date().setUTCHours(0, 0, 0, 0);
  const start = end - 30 * 24 * 60 * 60 * 1000;

  const usageOverTime = fillRange(
    usage.data.map(({ time, usage }) => ({ value: usage, time })),
    start,
    end,
  ).map(({ value, time }) => ({
    x: new Date(time).toUTCString(),
    y: value,
  }));

  const latestVerifications = await getLatestVerifications({ keyId: key.id });

  const totalUsage = usage.data.reduce((acc, { usage }) => acc + usage, 0);
  return (
    <div className="flex flex-col gap-8">
      <Card>
        <CardContent className="grid grid-cols-1 gap-px mx-auto divide-x sm:grid-cols-2 lg:grid-cols-4 ">
          <Stat label="Usage 30 days" value={totalUsage.toString()} />
          <Stat
            label="Expires"
            value={key.expires ? ms(key.expires.getTime() - Date.now()) : <Minus />}
          />
          <Stat label="Remaining" value={key.remainingRequests?.toString() ?? <Minus />} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Usage in the last 30 days</CardTitle>
          <CardDescription>See when this key was verified</CardDescription>
        </CardHeader>
        <CardContent>
          <ColumnChart data={usageOverTime} />
        </CardContent>
      </Card>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>Ratelimited</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {latestVerifications.data?.map((verification, i) => (
            // rome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
            <TableRow key={i}>
              <TableCell>
                <span className="font-medium text-content">
                  {new Date(verification.time).toUTCString()}
                </span>
              </TableCell>
              <TableCell>{verification.ratelimited ? <Badge>Ratelimited</Badge> : null}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

const Stat: React.FC<{ label: string; value: React.ReactNode }> = ({ label, value }) => (
  <div className="flex flex-wrap items-baseline justify-between px-4 py-2 gap-x-4 gap-y-2 sm:px-6 xl:px-8">
    <dt className="text-sm font-medium leading-6 text-content-subtle">{label}</dt>

    <dd className="flex-none w-full text-3xl font-medium leading-10 tracking-tight text-content">
      {value}
    </dd>
  </div>
);
