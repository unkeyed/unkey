import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { History } from "lucide-react";
export const revalidate = 0;

type Props = {
  params: {
    branchId: string;
  };
};

export default async function Page(props: Props) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return <div>Workspace with tenantId: {tenantId} not found</div>;
  }

  const branch = await db.query.gatewayBranches.findFirst({
    where: (table, { eq, and }) =>
      and(eq(table.workspaceId, workspace.id), eq(table.id, props.params.branchId)),
  });

  return (
    <div className="flex items-center justify-center">
      <Card className="max-w-md mx-auto h-min">
        <CardContent className="pt-6">
          <h2 className="text-lg font-semibold text-center mb-2">Deployment complete</h2>
          <p className="text-sm text-gray-600 text-center mb-6">
            Changes are deployed to <strong>{branch!.name}</strong> available at
            <br />
            <span className="underline text-blue-600">https://{branch!.domain}.unkey.app</span>.
          </p>
          <div className="flex justify-center w-full">
            <Button className="w-full ">
              <History className="mr-2 h-4 w-4" />
              Revert
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
