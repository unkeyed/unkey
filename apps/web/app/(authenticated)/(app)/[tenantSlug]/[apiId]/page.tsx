import { PageHeader } from "@/components/PageHeader";
import { CreateKeyButton } from "./CreateKey";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound } from "next/navigation";
import { KeyTable } from "./KeyTable";

type Props = {
  params: {
    tenantSlug: string;
    apiId: string;
  };
};
export default async function ApiOverviewPage(props: Props) {
  const _tenantId = getTenantId();
  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
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

      <KeyTable data={api.keys} />
    </div>
  );
}
