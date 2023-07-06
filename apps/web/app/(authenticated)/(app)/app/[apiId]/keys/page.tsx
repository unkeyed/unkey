import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq, type Key } from "@unkey/db";
import { notFound, redirect } from "next/navigation";
import { DeleteApiButton } from "../DeleteApi";
import { Separator } from "@/components/ui/separator";
import Link from "next/link";
import { ApiKeyTable } from "@/components/ApiKeyTable";
import { Badge } from "@/components/ui/badge";
import { CopyButton } from "@/components/CopyButton";
import { Button } from "@/components/ui/button";

export const revalidate = 0;
export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
    with: {
      workspace: true,
      keys: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return redirect("/onboarding");
  }

  const keys: Key[] = [];
  const expired: Key[] = [];

  for (const k of api.keys) {
    if (k.expires && k.expires.getTime() < Date.now()) {
      expired.push(k);
    } else {
      keys.push(k);
    }
  }
  if (expired.length > 0) {
    await Promise.all(expired.map((k) => db.delete(schema.keys).where(eq(schema.keys.id, k.id))));
  }

  return (
    <div>
      <ApiKeyTable data={keys} />
    </div>
  );
}
