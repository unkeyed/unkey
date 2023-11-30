import { StackedColumnChart } from "@/components/dashboard/charts";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { getDailyVerifications, getTotalActiveKeys } from "@/lib/tinybird";
import { fillRange } from "@/lib/utils";
import { sql } from "drizzle-orm";
import { redirect } from "next/navigation";

export const revalidate = 0;
export const runtime = "edge";

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
    .where(eq(schema.keys.keyAuthId, api.keyAuthId!))
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

  const usage = await getDailyVerifications({
    workspaceId: api.workspaceId,
    apiId: api.id,
  });

  const keys = await keysP;
  const active = await activeP;

  const successOverTime = fillRange(
    usage.data.map(({ time, success }) => ({ value: success, time })),
    start,
    end,
  ).map(({ value, time }) => ({
    x: new Date(time).toUTCString(),
    y: value,
  }));

  const ratelimitedOverTime = fillRange(
    usage.data.map(({ time, rateLimited }) => ({ value: rateLimited, time })),
    start,
    end,
  ).map(({ value, time }) => ({
    x: new Date(time).toUTCString(),
    y: value,
  }));

  const usageExceededOverTime = fillRange(
    usage.data.map(({ time, rateLimited }) => ({ value: rateLimited, time })),
    start,
    end,
  ).map(({ value, time }) => ({
    x: new Date(time).toUTCString(),
    y: value,
  }));

  const data = [
    ...successOverTime.map((d) => ({
      ...d,
      category: "Successful Verifications",
    })),
    ...ratelimitedOverTime.map((d) => ({ ...d, category: "Ratelimited" })),
    ...usageExceededOverTime.map((d) => ({ ...d, category: "Usage Exceeded" })),
  ];

  return (
    <div className="grid grid-cols-2 md:grid-cols-3 md:gap-4">
      <Card className="max-md:mb-4 max-md:mr-2 ">
        <CardHeader className="pb-6 ">
          <CardTitle>{formatNumber(keys)}</CardTitle>
          <CardDescription>Total Keys</CardDescription>
        </CardHeader>
      </Card>
      <Card className="max-md:mb-4 max-md:ml-2">
        <CardHeader className="pb-6">
          <CardTitle>
            {formatNumber(active.data.reduce((sum, day) => sum + day.usage, 0))}
          </CardTitle>
          <CardDescription>Active Keys (30 days)</CardDescription>
        </CardHeader>
      </Card>
      <Card className="col-span-2 max-md:mb-4 md:col-span-1">
        <CardHeader className="pb-6">
          <CardTitle>
            {formatNumber(
              usage.data.reduce((sum, day) => sum + day.success, 0),
            )}
          </CardTitle>
          <CardDescription>Successful Verifications (30 days)</CardDescription>
        </CardHeader>
      </Card>
      <Card className="relative col-span-3">
        <CardHeader>
          <CardTitle>Usage in the last 30 days</CardTitle>
          <CardDescription>
            This includes all key verifications in this API
          </CardDescription>
        </CardHeader>
        <CardContent>
          <StackedColumnChart data={data} />
        </CardContent>
      </Card>
    </div>
  );
}
