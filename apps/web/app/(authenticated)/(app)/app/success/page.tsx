import { PageHeader } from "@/components/dashboard/page-header";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { getQ1ActiveWorkspaces } from "@/lib/tinybird";
import { notFound } from "next/navigation";

async function getPaidCustomerCount() {
  const ws = await db.query.workspaces.findMany({
    where: (table, { and, eq, isNotNull }) =>
      and(eq(table.plan, "pro"), isNotNull(table.subscriptions)),
  });
  console.log(ws);

  return 10;
}

export default async function SuccessPage() {
  const activeWorkspaces = await getQ1ActiveWorkspaces({});
  const activeWorkspaceValue = activeWorkspaces.data[0];
  const date = new Date(activeWorkspaceValue.time);
  return (
    <div>
      <PageHeader title="Success Metrics" description="Unkey success metrics" />
      <h1 className="text-2xl font-semibold mb-8" />
      <Separator />
      <div className="flex flex-row gap-6">
        <Card className="flex w-72 mt-6">
          <div className="flex-col w-full py-4 px-6">
            <p className="text-sm text-content-subtle">Active Workspaces</p>
            <div className="text-2xl font-semibold leading-none tracking-tight mt-2">
              {activeWorkspaceValue.workspaces}
            </div>
            <div className=" text-sm text-content-subtle">{date.toDateString()}</div>
          </div>
        </Card>
        <Card className="flex w-72 mt-6">
          <div className="flex-col w-full py-4 px-6">
            <p className="text-sm text-content-subtle">Paying Customers</p>
            <div className="text-2xl font-semibold leading-none tracking-tight mt-2">
              {await getPaidCustomerCount()}
            </div>
            <div className=" text-sm text-content-subtle">{date.toDateString()}</div>

            {/* <div>JSON.stringify(activeWorkspaces.data, null, 2)}</div> */}
          </div>
        </Card>
      </div>
    </div>
  );
}
