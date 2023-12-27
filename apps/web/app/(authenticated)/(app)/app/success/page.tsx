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
    return <div>no stripe env</div>;
  }
  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2022-11-15",
    typescript: true,
  });
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });

  if (!workspace?.features.successPage) {
    return notFound();
  }

  const allInvoices: Stripe.Invoice[] = [];
  let hasMore = true;
  let startingAfter: string | undefined = undefined;
  while (hasMore) {
    await stripe.invoices
      .list({
        starting_after: startingAfter,
        status: "paid",
      })
      .then((res) => {
        allInvoices.push(...res.data);
        hasMore = res.has_more;
        startingAfter = res.data.at(-1)?.id;
      });
  }

  const billableInvoices = allInvoices.filter(
    (invoice) =>
      invoice.total > 0 && invoice.created >= Math.floor(Date.now() / 1000) - 45 * 24 * 60 * 60,
  );
  let customers = 0;
  const customerIds = new Set();
  billableInvoices.forEach((invoice) => {
    if (!customerIds.has(invoice.customer)) {
      customers += 1;
      customerIds.add(invoice.customer);
    }
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
              {customers}
            </div>
            <div className="mt-4">
              <Progress value={(customers / customerGoal) * 100} />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
