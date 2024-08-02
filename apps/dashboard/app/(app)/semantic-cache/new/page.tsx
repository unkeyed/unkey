import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { CreateLLMGatewayForm } from "../form";

export default async function NewSemanticCachePage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
    with: {
      llmGateways: {
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (workspace?.llmGateways.length) {
    return redirect(`/semantic-cache/${workspace.llmGateways[0].id}/logs`);
  }

  return (
    <div>
      <CreateLLMGatewayForm />
    </div>
  );
}
