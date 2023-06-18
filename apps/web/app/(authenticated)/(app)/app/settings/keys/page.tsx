import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound } from "next/navigation";
import { Row } from "./row";
import { CreateKeyButton } from "./CreateKey";
import { Separator } from "@/components/ui/separator";

export default async function SettingsKeysPage() {
  const workspaceId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.id, workspaceId),
    with: {
      keys: {
        where: eq(schema.keys.internal, true),
      },
    },
  });
  if (!workspace) {
    return notFound();
  }
  return (
    <div>
      <PageHeader
        title="Keys"
        description="Manage your own API keys"
        actions={[<CreateKeyButton key="create-key" />]}
      />
      <Separator className="my-6" />
      {tenant.keys.length === 0 ? (
        ""
      ) : (
        <ul role="list" className="mt-8 divide-y divide-white/10">
          {workspace.keys
            .sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime())
            .map((apiKey) => (
              <Row key={apiKey.id} apiKey={apiKey} />
            ))}
        </ul>
      )
      }
    </div >
  );
}
