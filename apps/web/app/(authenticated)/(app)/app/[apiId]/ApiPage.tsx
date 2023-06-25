import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@/lib/db";
import { redirect } from "next/navigation";
import { getActiveCount, getUsage, Tinybird } from "@unkey/tinybird";
import { env } from "@/lib/env";
import { fillRange } from "@/lib/utils";
import { ColumnChart } from "@/components/charts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { sql } from "drizzle-orm";

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

  const activeP = getActiveCount(new Tinybird({ token: env.TINYBIRD_TOKEN }))({
    workspaceId: api.workspaceId,
    apiId: api.id,
  });

  const usage = await getUsage(new Tinybird({ token: env.TINYBIRD_TOKEN }))({
    workspaceId: api.workspaceId,
    apiId: api.id,
  });
  const now = new Date();
  const year = now.getUTCFullYear();
  const month = now.getUTCMonth();
  const start = new Date(year, month, 0, 0, 0, 0, 0);
  const end = new Date(start.getTime());
  end.setUTCMonth(end.getUTCMonth() + 1);

  const keys = await keysP;
  const active = await activeP;

  const usageOverTime = fillRange(
    usage.data.map(({ time, usage }) => ({ value: usage, time })),
    start.getTime(),
    end.getTime(),
    "1d",
  ).map(({ value, time }) => ({
    x: new Date(time).toUTCString(),
    y: value,
  }));

  return (
    <div className="grid grid-cols-3 gap-4">
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle>{keys}</CardTitle>
          <CardDescription>Total Keys</CardDescription>
        </CardHeader>
      </Card>
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle>{active.data.at(0)?.active ?? 0}</CardTitle>
          <CardDescription>Active Keys (30 days)</CardDescription>
        </CardHeader>
      </Card>
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle>{usage.data.reduce((sum, day) => sum + day.usage, 0)}</CardTitle>
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
