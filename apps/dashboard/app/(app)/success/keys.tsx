import { AreaChart } from "@/components/dashboard/charts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { notFound } from "next/navigation";

export const revalidate = 60;

export async function Keys() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });

  if (!workspace?.features.successPage) {
    return notFound();
  }

  let keys = await db.query.keys.findMany({
    where: (table, { eq, not }) => not(eq(table.workspaceId, env().UNKEY_WORKSPACE_ID)),
  });

  keys = keys
    .filter((r) => r.createdAt !== null)
    .sort((a, b) => a.createdAt!.getTime() - b.createdAt!.getTime());
  let sum = 0;
  const chartData = Object.entries(
    keys.reduce(
      (acc, role) => {
        sum += 1;
        const startOfDay = role.createdAt!;
        startOfDay.setUTCHours(0, 0, 0, 0);
        const t = startOfDay.getTime().toString();

        acc[t] = sum;
        return acc;
      },
      {} as Record<string, number>,
    ),
  ).map(([t, y]) => ({
    x: new Date(Number.parseInt(t)).toLocaleDateString(),
    y,
  }));

  return (
    <Card className="flex flex-col w-full h-fit">
      <CardHeader>
        <CardTitle>Keys</CardTitle>
        <CardDescription>How many keys exist</CardDescription>
      </CardHeader>
      <CardContent>
        <AreaChart data={chartData} timeGranularity="day" tooltipLabel="Total keys" />
      </CardContent>
    </Card>
  );
}
