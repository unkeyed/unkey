import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { CreateSecretForm } from "./form";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function SecretsPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq, isNull, and }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return notFound();
  }

  return (
    <>
      <PageHeader
        title="Create a new secret"
        description="Secrets are encrypted using AES-GCM"
        actions={[]}
      />
      <CreateSecretForm />
    </>
  );
}
