import { getAuthOrRedirect } from "@/lib/auth";
import { db } from "@/lib/db";
import { WorkspaceNavbar } from "../workspace-navbar";
import { TeamPageClient } from "./client";

export const revalidate = 0;

export default async function SettingTeamPage() {
  const { orgId } = await getAuthOrRedirect();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: { quotas: true },
  });

  const team = workspace?.quotas?.team ?? false;

  return workspace ? (
    <>
      <WorkspaceNavbar
        workspace={{ id: workspace.id, name: workspace.name }}
        activePage={{ href: "team", text: "Team" }}
      />
      <div className="flex flex-col items-center justify-center w-full px-16 gap-4 mt-10">
        <TeamPageClient team={team} />
      </div>
    </>
  ) : (
    <div>
      <div>Workspace not found</div>
    </div>
  );
}
