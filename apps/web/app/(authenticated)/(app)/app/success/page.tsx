import { AreaChart } from "@/components/dashboard/charts";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { getQ1ActiveWorkspaces } from "@/lib/tinybird";
import { notFound } from "next/navigation";
import Stripe from "stripe";
export const revalidate = 60;

export default async function SuccessPage() {
  const e = stripeEnv();
  if (!e) {
    return <div>Stripe env not set</div>;
  }

  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });

  if (!workspace?.features.successPage) {
    return notFound();
  }

  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2022-11-15",
    typescript: true,
  });

  const subscriptions = await stripe.subscriptions.list({
    status: "active",
  });

  const activeWorkspaces = await getQ1ActiveWorkspaces({});
  const chartData = activeWorkspaces.data.map(({ time, workspaces }) => ({
    x: new Date(time).toLocaleDateString(),
    y: workspaces,
  }));
  const customerGoal = 7;
  const activeWorkspaceGoal = 300;
  return (
    <div>
      <div className="w-full">
        <PageHeader title="Success Metrics Q1" />
        <Separator />
      </div>
      <div className="flex gap-6 p-6 w-full flex-col">
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
              {subscriptions.data.length}
            </div>
            <div className="mt-4">
              <Progress value={(subscriptions.data.length / customerGoal) * 100} />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
