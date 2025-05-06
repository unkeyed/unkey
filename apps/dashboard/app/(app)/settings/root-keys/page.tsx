import { PageHeader } from "@/components/dashboard/page-header";
import { RootKeyTable } from "@/components/dashboard/root-key-table";
import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { WorkspaceNavbar } from "../workspace-navbar";

export const revalidate = 0;

export default async function SettingsKeysPage(_props: {
  params: { apiId: string };
}) {
  const { orgId } = await getAuth();

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
      <WorkspaceNavbar
        workspace={workspace}
        activePage={{ href: "root-keys", text: "Root Keys" }}
      />
      <div className="flex flex-col items-center justify-center w-full px-6 gap-4">
        <PageHeader
          className="mt-4"
          title="Root Keys"
          description="Root keys are used to interact with the Unkey API."
        />
        <div className="grid w-full grid-cols-1 gap-8 mb-20">
          <RootKeyTable data={keys} />
        </div>
      </div>
    </div>
  );
}
