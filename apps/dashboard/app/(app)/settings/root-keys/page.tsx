import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { RootKeyTable } from "@/components/dashboard/root-key-table";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Gear } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { redirect } from "next/navigation";
import { navigation } from "../constants";

export const revalidate = 0;

export default async function SettingsKeysPage(_props: {
  params: { apiId: string };
}) {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });
  if (!workspace) {
    return redirect("/new");
  }

  const keys = await db.query.keys.findMany({
    where: (table, { eq, and, or, isNull, gt }) =>
      and(
        eq(table.forWorkspaceId, workspace.id),
        isNull(table.deletedAtM),
        or(isNull(table.expires), gt(table.expires, new Date())),
      ),
    limit: 100,
  });

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gear />}>
          <Navbar.Breadcrumbs.Link href="/settings/root-keys" active>
            Settings
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Link key="create-root-key" href="/settings/root-keys/new">
            <Button variant="primary">Create New Root Key</Button>
          </Link>
        </Navbar.Actions>
      </Navbar>
      <PageContent>
        <SubMenu navigation={navigation} segment="root-keys" />

        <PageHeader
          className="mt-8"
          title="Root Keys"
          description="Root keys are used to interact with the Unkey API."
        />
        <div className="mb-20 grid w-full grid-cols-1 gap-8">
          <RootKeyTable data={keys} />
        </div>
      </PageContent>
    </div>
  );
}
