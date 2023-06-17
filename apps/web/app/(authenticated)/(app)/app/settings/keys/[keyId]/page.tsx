import { notFound, redirect } from "next/navigation";
import { db, schema, eq, and } from "@unkey/db";

import { getTenantId } from "@/lib/auth";
import { Client } from "./client";

export default async function Page(props: { params: { keyId: string } }) {
  const workspaceId = getTenantId();
  console.log({ props });
  const apiKey = await db.query.keys.findFirst({
    where: and(eq(schema.keys.workspaceId, workspaceId), eq(schema.keys.id, props.params.keyId)),
  });
  console.log({ apiKey });
  if (!apiKey) {
    return notFound();
  }
  if (apiKey.workspaceId !== workspaceId) {
    return notFound();
  }

  return <Client apiKey={apiKey} />;
}
