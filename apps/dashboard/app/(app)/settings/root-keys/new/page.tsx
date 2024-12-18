import { PageHeader } from "@/components/dashboard/page-header";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Gear } from "@unkey/icons";
import { redirect } from "next/navigation";
import { Client } from "./client";

export const revalidate = 0;

export default async function SettingsKeysPage(_props: {
  params: { apiId: string };
}) {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gear />}>
          <Navbar.Breadcrumbs.Link href="/settings/root-keys">Root Keys</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link active href="/settings/root-keys/new">
            Create new key
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
      <PageContent>
        <PageHeader
          title="Create a new Root Key"
          description="Select the permissions you want to grant to your new api key and click the button below to create it."
        />

        <Client apis={workspace.apis} />
      </PageContent>
    </div>
  );
}
