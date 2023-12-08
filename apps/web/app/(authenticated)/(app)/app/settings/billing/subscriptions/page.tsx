/**
 * This page is only for debugging purposes
 */
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";

export const revalidate = 0;

export default async function PlanPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Plan</CardTitle>
        <CardDescription>For debugging purposes</CardDescription>
      </CardHeader>
      <CardContent>
        <Code>
          {JSON.stringify(
            {
              workspaceId: workspace.id,
              stripeCustomerId: workspace.stripeCustomerId,
              subscriptions: workspace.subscriptions,
            },
            null,
            2,
          )}
        </Code>
      </CardContent>
    </Card>
  );
}
