import { PageHeader } from "@/components/PageHeader";
import { CreateKeyButton } from "./CreateKey";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq, and } from "@unkey/db";
import { notFound, redirect } from "next/navigation";
import { KeyTable } from "./KeyTable";

type Props = {
  params: {
    apiId: string;
  };
};
export default async function ApiOverviewPage(props: Props) {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: {
        where: eq(schema.apis.id, props.params.apiId),
        with: {
          keys: true,
        },
      },
    },
  });
  if (!workspace) {
    return redirect("/onboarding");
  }
  const api = workspace.apis.at(0);
  if (!api) {
    return notFound();
  }
  return (
    <div>
      <PageHeader
        title={api.name}
        description={api.id}
        actions={[<CreateKeyButton key="createKey" apiId={api.id} />]}
      />

      <KeyTable data={api.keys.sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime())} />
    </div>
  );
}
