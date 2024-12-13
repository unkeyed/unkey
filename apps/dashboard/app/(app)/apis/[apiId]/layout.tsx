import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import type { PropsWithChildren } from "react";

type Props = PropsWithChildren<{
  params: {
    apiId: string;
  };
}>;

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPageLayout(props: Props) {
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

  return props.children;
}
