import { PageContent } from "@/components/page-content";
import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { WorkspaceNavbar } from "../workspace-navbar";
import TeamPageClient from "./client";

export const revalidate = 0;

export default async function SettingTeamPage() {
  const orgId = await getOrgId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: { quotas: true },
  });

  const team = workspace?.quotas?.team ?? false;

  return workspace ? (
    <div>
      <WorkspaceNavbar
        workspace={{ id: workspace.id, name: workspace.name }}
        activePage={{ href: "team", text: "Team" }}
      />
      <PageContent>
        <div className="flex flex-col gap-8 mt-8 mb-20">
          <TeamPageClient team={team} />
        </div>
      </PageContent>
    </div>
  ) : (
    <div>
      <div>Workspace not found</div>
    </div>
  );
}
