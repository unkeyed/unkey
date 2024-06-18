import { CopyButton } from "@/components/dashboard/copy-button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { DeleteGateway } from "./delete-gateway";

export default async function SemanticCacheSettingPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      llmGateways: {
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  if (!workspace.llmGateways.length) {
    return redirect("/semantic-cache/new");
  }

  const gateway = workspace.llmGateways[0];

  console.info("gateway", gateway);

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Gateway Name</CardTitle>
        </CardHeader>
        <CardContent>
          <Code className="flex items-center justify-between w-full h-8 max-w-sm gap-4 cursor-pointer">
            <pre>{gateway.name}</pre>
            <div className="flex items-start justify-between gap-4">
              <CopyButton value={gateway.name} />
            </div>
          </Code>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Gateway ID</CardTitle>
          <CardDescription>This is your gateway ID. It's used in some API calls.</CardDescription>
        </CardHeader>
        <CardContent>
          <Code className="flex items-center justify-between w-full sm:h-8 max-w-sm gap-4 cursor-pointer">
            <pre className="whitespace-normal" style={{ wordBreak: "break-word" }}>
              {" "}
              {gateway.id}
            </pre>
            <div className="flex items-start justify-between gap-4">
              <CopyButton value={gateway.id} />
            </div>
          </Code>
        </CardContent>
      </Card>
      <DeleteGateway gateway={gateway} />
    </div>
  );
}
