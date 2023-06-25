import { db, eq, schema } from "@/lib/db";
import { notFound } from "next/navigation";

import { getTenantId } from "@/lib/auth";
import { Client } from "./client";

export default async function Page(props: { params: { keyId: string } }) {
  const tenantId = getTenantId();
  const apiKey = await db.query.keys.findFirst({
    where: eq(schema.keys.id, props.params.keyId),
    with: {
      api: {
        with: {
          workspace: true,
        },
      },
    },
  });
  if (!apiKey) {
    return notFound();
  }
  if (apiKey.api?.workspace?.tenantId !== tenantId) {
    return notFound();
  }

  return <Client apiKey={apiKey} />;
}
