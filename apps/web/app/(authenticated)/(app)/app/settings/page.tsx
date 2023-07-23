import { PageHeader } from "@/components/dashboard/page-header";
import { ColumnChart } from "@/components/dashboard/charts";
import { Text } from "@/components/dashboard/text";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { fillRange } from "@/lib/utils";
import { db, eq, schema } from "@/lib/db";
import { getDailyUsage } from "@/lib/tinybird";
import { redirect } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { CopyButton } from "@/components/dashboard/copy-button";

export const revalidate = 0;

export default async function SettingsPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/onboarding");
  }
  const usage = await getDailyUsage({
    workspaceId: workspace.id,
  });

  const now = new Date();
  const year = now.getUTCFullYear();
  const month = now.getUTCMonth();
  const start = new Date(year, month, 0, 0, 0, 0, 0);
  const end = new Date(start.getTime());
  end.setUTCMonth(end.getUTCMonth() + 1);

  const usageOverTime = fillRange(
    usage.data.map(({ time, usage }) => ({ value: usage, time })),
    start.getTime(),
    end.getTime(),
  ).map(({ value, time }) => ({
    x: new Date(time).toUTCString(),
    y: value,
  }));

  const totalUsage = usage.data.reduce((sum, day) => {
    return sum + day.usage;
  }, 0);
  return (
    <div>
      <PageHeader
        title={workspace.name}
        description="Settings"
        actions={[
          <Badge key="workspaceId" variant="outline" className="font-mono font-medium">
            {workspace.id}
            <CopyButton value={workspace.id} className="ml-2" />
          </Badge>,
        ]}
      />

      <Card>
        <CardHeader>
          <CardTitle>Usage & Billing</CardTitle>
          <CardDescription>
            <Text>
              Current billing cycle:{" "}
              <strong>
                {new Date(year, month, 1).toLocaleString(undefined, {
                  month: "long",
                })}{" "}
                {year}
              </strong>{" "}
            </Text>
          </CardDescription>
        </CardHeader>

        <CardContent>
          <div className="flex justify-center py-4 divide-x divide-zinc-200">
            <div className="flex flex-col items-center gap-2 px-8">
              <Text size="xl">Current Usage</Text>
              <div className="flex items-center gap-2">
                <Text size="lg">{totalUsage.toLocaleString()}</Text>/{" "}
                <svg
                  className="w-8 h-8"
                  fill="none"
                  shapeRendering="geometricPrecision"
                  stroke="currentColor"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="1.5"
                  viewBox="0 0 24 24"
                  width="24"
                  height="24"
                >
                  <title>usage</title>
                  <path d="M13.833 8.875S15.085 7 18.043 7C21 7 23 9.5 23 12s-1.784 5-4.864 5-4.914-3.124-6.136-5c-1.222-1.875-3.392-5-6.446-5S1 9.5 1 12s1.351 5 4.648 5c3.296 0 4.519-1.875 4.519-1.875" />
                </svg>
              </div>
            </div>
          </div>
          <div className="p-8">
            <div className="h-48">
              <ColumnChart data={usageOverTime} />
            </div>
          </div>
        </CardContent>
        {/* <CardFooter className="flex items-center justify-end gap-2">
                    <BillingButton teamId={team.id} />
                    <Link key="plans" href={`/${team.slug}/settings/plans`}>
                        <Button variant="primary">Change your plan</Button>
                    </Link>
                </CardFooter> */}
      </Card>
    </div>
  );
}
