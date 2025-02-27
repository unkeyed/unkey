import { ColumnChart } from "@/components/dashboard/charts";
import { Loading } from "@/components/dashboard/loading";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { and, count, db, gte, isNotNull, schema, sql } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import Stripe from "stripe";
import { Chart } from "./chart";

export const revalidate = 60;
export const dynamic = "force-dynamic";

export default async function SuccessPage() {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });

  if (!workspace?.features.successPage) {
    return notFound();
  }
  let customers = 0;
  const allInvoices: Stripe.Invoice[] = [];
  const e = stripeEnv();

  if (e) {
    const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
      apiVersion: "2023-10-16",
      typescript: true,
    });

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
    const customerIds = new Set();
    billableInvoices.forEach((invoice) => {
      if (!customerIds.has(invoice.customer)) {
        customers += 1;
        customerIds.add(invoice.customer);
      }
    });
  }

  const activeWorkspaces = await clickhouse.business.activeWorkspaces().then((res) => res.val!);
  const chartData = activeWorkspaces.map(({ time, workspaces }) => ({
    x: new Date(time).toLocaleDateString(),
    y: workspaces,
  }));

  const tables = {
    Workspaces: schema.workspaces,
    Apis: schema.apis,
    Keys: schema.keys,
    Permissions: schema.permissions,
    Roles: schema.roles,
    "Ratelimit Namespaces": schema.ratelimitNamespaces,
    "Ratelimit Overrides": schema.ratelimitOverrides,
  };

  const t0 = new Date("2024-01-01").getTime();
  return (
    <div>
      <div className="w-full">
        <PageHeader title="Success Metrics" description="Unkey success metrics" />
        <div className="mb-8 text-2xl font-semibold" />
        <Separator />
      </div>
      <div className="grid w-full grid-cols-2 gap-6 p-6">
        <Card>
          <CardHeader>
            <CardTitle>Active Workspaces</CardTitle>
          </CardHeader>
          <CardContent className="relative h-40">
            <ColumnChart
              padding={[8, 40, 64, 40]}
              data={chartData}
              timeGranularity="month"
              tooltipLabel="Active Workspaces"
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Paying Customers</CardTitle>
            <CardDescription>Delayed by up to a month</CardDescription>
          </CardHeader>
          <CardContent className="relative h-40">
            <div className="mt-2 text-2xl font-semibold leading-none tracking-tight">
              {customers}
            </div>
            <div className="mt-4">
              <Progress value={(customers / 50) * 100} />
            </div>
          </CardContent>
        </Card>
        {Object.entries(tables).map(([title, table]) => (
          <Suspense fallback={<Loading />} key={title}>
            <Chart
              key={title}
              title={title}
              t0={t0}
              query={() =>
                db
                  .select({
                    date: sql<string>`DATE(created_at) as date`,
                    count: count(),
                  })
                  .from(table)
                  .where(and(isNotNull(table.createdAtM), gte(table.createdAtM, t0)))
                  .groupBy(sql`date`)
                  .orderBy(sql`date ASC`)
                  .execute()
              }
            />
          </Suspense>
        ))}
      </div>
    </div>
  );
}
