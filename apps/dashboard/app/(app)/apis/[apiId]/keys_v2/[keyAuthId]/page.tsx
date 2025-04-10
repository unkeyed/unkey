import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { KeysClient } from "./_components/keys-client";
import { Navigation } from "./navigation";

export const dynamic = "force-dynamic";

export default async function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const orgId = await getOrgId();

  const keyAuth = await db.query.keyAuth.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.keyAuthId), isNull(table.deletedAtM)),
    with: {
      workspace: true,
      api: true,
    },
  });
  if (!keyAuth || keyAuth.workspace.orgId !== orgId) {
    return notFound();
  }

  return (
    <div>
      <Navigation apiId={props.params.apiId} keyAuth={keyAuth} />
      <KeysClient keyspaceId={keyAuth.id} />
    </div>
  );
}
