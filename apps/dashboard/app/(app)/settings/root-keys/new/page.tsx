import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { Client } from "./client";

export const revalidate = 0;

export default async function SettingsKeysPage(_props: {
  params: { apiId: string };
}) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
    with: {
      apis: {},
    },
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div className="min-h-screen ">
      <PageHeader
        title="Create a new Root Key"
        description="Select the permissions you want to grant to your new api key and click the button below to create it."
      />

      <Client apis={workspace.apis} />
    </div>
  );
}
