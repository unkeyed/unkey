import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import type { PropsWithChildren } from "react";

type Props = PropsWithChildren<{
  params: {
    namespaceId: string;
  };
}>;

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RatelimitNamespacePageLayout(props: Props) {
  const tenantId = getTenantId();

  const namespace = await db.query.ratelimitNamespaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.namespaceId), isNull(table.deletedAt)),
    with: {
      workspace: true,
    },
  });

  if (!namespace || namespace.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return props.children;
}
