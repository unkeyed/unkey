import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { KeysClient } from "./_components/keys-client";
import { Navigation } from "./navigation";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const tenantId = getTenantId();
  const keyAuth = await db.query.keyAuth.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.keyAuthId), isNull(table.deletedAtM)),
    with: {
      workspace: true,
      api: true,
    },
  });

  if (!keyAuth || keyAuth.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div>
      <Navigation apiId={props.params.apiId} keyA={keyAuth} />
      <KeysClient keyspaceId={keyAuth.id} />
    </div>
  );
}
