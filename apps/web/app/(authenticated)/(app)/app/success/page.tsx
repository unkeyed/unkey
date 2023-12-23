import { AreaChart } from "@/components/dashboard/charts";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNotNull, not, schema, sql } from "@/lib/db";
import { getQ1ActiveWorkspaces } from "@/lib/tinybird";
import { notFound } from "next/navigation";

export const revalidate = 60;

export default async function SuccessPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });

  if (!workspace?.features.successPage) {
    return notFound();
  }

  const payingWorkspaces = await db
    .select({ count: sql<string>`count(*)` })
    .from(schema.workspaces)
    .where(
      and(
        isNotNull(schema.workspaces.subscriptions),
        not(eq(schema.workspaces.plan, "free")),
        isNotNull(schema.workspaces.subscriptions),
      ),
    )
    .then((rows) => {
      const count = rows.at(0)?.count;
      return count ? parseInt(count) : 0;
    });

  const activeWorkspaces = await getQ1ActiveWorkspaces({});
  const chartData = activeWorkspaces.data.map(({ time, workspaces }) => ({
    x: new Date(time).toLocaleDateString(),
    y: workspaces,
  }));
  const customerGoal = 6;
  const activeWorkspaceGoal = 300;
  return (
    <div>
      <div className="w-full">
        <PageHeader title="Success Metrics" description="Unkey success metrics" />
        <h1 className="text-2xl font-semibold mb-8" />
        <Separator />
      </div>
      <div className="flex gap-6 p-6 w-full">
        <Card className="w-full">
          <CardHeader>
            <CardTitle>Active Workspaces</CardTitle>
            <CardDescription>{`Current goal of ${activeWorkspaceGoal}`}</CardDescription>
          </CardHeader>
          <CardContent>
            <div>
              <AreaChart data={chartData} timeGranularity="day" tooltipLabel="Active Workspaces" />
            </div>
          </CardContent>
        </Card>

        <Card className="flex flex-col h-fit w-full">
          <CardHeader>
            <CardTitle>Paying Customers</CardTitle>
            <CardDescription>Current goal of {customerGoal}</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold leading-none tracking-tight mt-2">
              {payingWorkspaces}
            </div>
            <div className="mt-4">
              <Progress value={(payingWorkspaces / customerGoal) * 100} />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
