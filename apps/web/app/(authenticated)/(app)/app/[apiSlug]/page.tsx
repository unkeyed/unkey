import { PageHeader } from "@/components/PageHeader";
import { CreateKeyButton } from "./CreateKey";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq, and } from "@unkey/db";
import { notFound } from "next/navigation";
import { KeyTable } from "./KeyTable";

type Props = {
  params: {
    tenantSlug: string;
    apiSlug: string;
  };
};
export default async function ApiOverviewPage(props: Props) {
  const tenantId = getTenantId();
  const api = await db.query.apis.findFirst({
    where: and(eq(schema.apis.slug, props.params.apiSlug), eq(schema.apis.tenantId, tenantId)),
    with: {
      keys: true,
    },
  });
  if (!api) {
    return notFound();
  }
  console.log(api);
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
