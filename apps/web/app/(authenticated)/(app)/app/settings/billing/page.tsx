import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { type Workspace, db, eq, schema } from "@/lib/db";
import Link from "next/link";
import { redirect } from "next/navigation";

export const revalidate = 0;

export default async function BillingPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/onboarding");
  }

  return (
    <div className="flex flex-col gap-8 mb-20 ">
      <Card>
        <CardHeader>
          <CardTitle>Current Plan</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2">
            <input type="hidden" name="workspaceId" value={workspace.id} />
            <label className="hidden sr-only">Name</label>
            <div className="flex flex-col items-center justify-center gap-2">
              <Plan plan={workspace.plan} />
              <p className="text-xs text-content-subtle">
                Your organization is currently on the {workspace.plan} plan.
              </p>
            </div>
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Link href="/app/settings/billing/stripe">
            <Button>Change Plan</Button>
          </Link>
        </CardFooter>
      </Card>
    </div>
  );
}

const Plan: React.FC<{ plan: Workspace["plan"] }> = ({ plan }) => {
  switch (plan) {
    case "free":
      return <Badge variant="secondary">Free</Badge>;
    case "pro":
      return <Badge>Pro</Badge>;
    case "enterprise":
      return <Badge variant="primary">Enterprise</Badge>;
  }
};
