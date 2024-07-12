import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { faker } from "@faker-js/faker";
import { redirect } from "next/navigation";
import { CreateLLMGatewayForm } from "../form";
import { generateSemanticCacheDefaultName } from "./util/generate-semantic-cache-default-name";

export default async function NewSemanticCachePage() {
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

  if (workspace?.llmGateways.length) {
    return redirect(`/semantic-cache/${workspace.llmGateways[0].id}/logs`);
  }

  const defaultName = generateSemanticCacheDefaultName();

  return (
    <div>
      <CreateLLMGatewayForm defaultName={defaultName} />
    </div>
  );
}
