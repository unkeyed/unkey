import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { CreateKey } from "./client";

export const dynamic = "force-dynamic";
export default async function ApiPage(props: { params: { keyAuthId: string } }) {
  const tenantId = getTenantId();

  console.log("XXXX");
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
    <div>
      <CreateKey keyAuthId={keyAuth.id} />
    </div>
  );
}
