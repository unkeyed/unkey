import { notFound, redirect } from "next/navigation";
import { db, schema, eq, and } from "@unkey/db";

import { getTenantId } from "@/lib/auth";
import { Client } from "./client";

export default async function Page(props: { params: { keyId: string } }) {
  const tenantId = getTenantId();
  console.log({ props });
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
  console.log({ apiKey });
  if (!apiKey) {
    return notFound();
  }
  if (apiKey.api?.workspace?.tenantId !== tenantId) {
    return notFound();
  }

  return <Client apiKey={apiKey} />;
}
