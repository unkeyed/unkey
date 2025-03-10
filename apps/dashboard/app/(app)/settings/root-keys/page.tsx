import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { RootKeyTable } from "@/components/dashboard/root-key-table";
import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { navigation } from "../constants";
import { Navigation } from "./navigation";

export const revalidate = 0;

export default async function SettingsKeysPage(_props: {
  params: { apiId: string };
}) {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
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
      <Navigation />
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
