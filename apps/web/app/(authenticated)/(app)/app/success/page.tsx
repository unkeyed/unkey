import { AreaChart } from "@/components/dashboard/charts";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { db } from "@/lib/db";
import { getQ1ActiveWorkspaces } from "@/lib/tinybird";

async function getPaidCustomerCount() {
  const ws = await db.query.workspaces.findMany({
    where: (table, { and, eq, isNotNull, not }) =>
      and(isNotNull(table.subscriptions), not(eq(table.plan, "free"))),
  });
  return ws.length;
}

function getChartData(data: any[]) {
  const chartData = data.map((element) => {
    return {
      x: new Date(element.time).toLocaleDateString(),
      y: element.workspaces,
    };
  });
  return chartData;
}

export default async function SuccessPage() {
  const activeWorkspaces = await getQ1ActiveWorkspaces({});
  const chartData = getChartData(activeWorkspaces.data);
  const activeWorkspaceValue = activeWorkspaces.data[0];
  const date = new Date(activeWorkspaceValue.time);
  const paidCustomers = await getPaidCustomerCount();
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
            <CardDescription>Current goal of 6</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold leading-none tracking-tight mt-2">
              {paidCustomers}
            </div>
            <div className="mt-4">
              <Progress value={(paidCustomers / customerGoal) * 100} />
            </div>

            <div className=" text-sm text-content-subtle mt-6">{date.toDateString()}</div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
