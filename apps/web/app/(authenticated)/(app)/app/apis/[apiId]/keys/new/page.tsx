import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { CreateKey } from "../../create-key";

export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.apiId), isNull(table.deletedAt)),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div className="h-full">
      <CreateKey apiId={api.id} />
    </div>
  );
}
