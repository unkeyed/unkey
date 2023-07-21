import { ColumnChart } from "@/components/dashboard/charts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { fillRange } from "@/lib/utils";
import { db, eq, schema } from "@unkey/db";
import { getTotalActiveKeys, getDailyUsage } from "@/lib/tinybird";
import { sql } from "drizzle-orm";
import { redirect } from "next/navigation";
import { formatNumber } from "@/lib/fmt";

export const revalidate = 0;

export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return redirect("/onboarding");
  }

  const keysP = db
    .select({ count: sql<number>`count(*)` })
    .from(schema.keys)
    .where(eq(schema.keys.apiId, api.id))
    .execute()
    .then((res) => res.at(0)?.count ?? 0);

  const end = new Date().setUTCHours(0, 0, 0, 0);
  // start of the day 30 days ago
  const start = end - 30 * 24 * 60 * 60 * 1000;
  const activeP = getTotalActiveKeys({
    workspaceId: api.workspaceId,
    apiId: api.id,
    start,
    end,
  });

  const usage = await getDailyUsage({
    workspaceId: api.workspaceId,
    apiId: api.id,
  });

  const keys = await keysP;
  const active = await activeP;
  console.log(
    JSON.stringify({ workspaceId: api.workspace.id, apiId: api.id, start, end, active }, null, 2),
  );

  const usageOverTime = fillRange(
    usage.data.map(({ time, usage }) => ({ value: usage, time })),
    start,
    end,
  ).map(({ value, time }) => ({
    x: new Date(time).toUTCString(),
    y: value,
  }));

  return (
    <div className="grid grid-cols-3 gap-4">
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle>{formatNumber(keys)}</CardTitle>
          <CardDescription>Total Keys</CardDescription>
        </CardHeader>
      </Card>
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle>
            {formatNumber(active.data.reduce((sum, day) => sum + day.usage, 0))}
          </CardTitle>
          <CardDescription>Active Keys (30 days)</CardDescription>
        </CardHeader>
      </Card>
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle>{formatNumber(usage.data.reduce((sum, day) => sum + day.usage, 0))}</CardTitle>
          <CardDescription>Verifications (30 days)</CardDescription>
        </CardHeader>
      </Card>
      <Card className="col-span-3">
        <CardHeader>
          <CardTitle>Usage in the last 30 days</CardTitle>
          <CardDescription>This includes all key verifications in this API</CardDescription>
        </CardHeader>
        <CardContent>
          <ColumnChart data={usageOverTime} />
        </CardContent>
      </Card>
    </div>
  );
}
