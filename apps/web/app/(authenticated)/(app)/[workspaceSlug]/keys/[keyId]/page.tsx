import { db, eq, schema } from "@/lib/db";
import { notFound } from "next/navigation";

import { getTenantId } from "@/lib/auth";
import { Client } from "./client";

export default async function Page(props: { params: { keyId: string } }) {
  const tenantId = getTenantId();
  const apiKey = await db.query.keys.findFirst({
    where: eq(schema.keys.id, props.params.keyId),
  });
  if (!apiKey) {
    return notFound();
  }
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.id, apiKey.workspaceId),
  });
  if (!workspace) {
    return notFound();
  }
  if (workspace.tenantId !== tenantId) {
    return notFound();
  }

  return <Client apiKey={apiKey} />;
}
