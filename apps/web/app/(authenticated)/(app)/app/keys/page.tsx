import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound, redirect } from "next/navigation";
import { Row } from "./row";
import { CreateKeyButton } from "./CreateKey";
import { Separator } from "@/components/ui/separator";
import { env } from "@/lib/env";

export default async function SettingsKeysPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/onboarding");
  }

  const keys = await db.query.keys.findMany({
    where: eq(schema.keys.forWorkspaceId, workspace.id),
  });

  return (
    <div>
      <PageHeader
        title="Keys"
        description="These keys are used to interact with the unkey API"
        actions={[<CreateKeyButton key="create-key" />]}
      />
      <Separator className="my-6" />

      {keys.length === 0 ? (
        "No keys here, did you look in the fridge?"
      ) : (
        <ul role="list" className="mt-8 divide-y divide-white/10">
          {keys
            .sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime())
            .map((apiKey) => (
              <Row key={apiKey.id} apiKey={apiKey} />
            ))}
        </ul>
      )}
    </div>
  );
}
