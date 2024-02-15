import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { Keys } from "./keys";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPage(props: { params: { apiId: string; keyAuthId: string } }) {
  const tenantId = getTenantId();

  const keyAuth = await db.query.keyAuth.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.keyAuthId), isNull(table.deletedAt)),
    with: {
      workspace: true,
    },
  });
  if (!keyAuth || keyAuth.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div className="flex flex-col gap-8 mb-20 ">
      <Keys keyAuthId={keyAuth.id} />
    </div>
  );
}
