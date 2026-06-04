import { PageChrome } from "@/components/page-header/page-chrome";
import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { WorkspaceNavbar } from "../workspace-navbar";
import { TeamPageClient } from "./client";

export const revalidate = 0;

export default async function SettingTeamPage() {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: { quotas: true },
  });

  const team = workspace?.quotas?.team ?? false;

  const header = (
    <PageHeader>
      <PageHeaderContent>
        <PageHeaderTitle>Team</PageHeaderTitle>
      </PageHeaderContent>
    </PageHeader>
  );
  const legacyHeader = <WorkspaceNavbar activePage={{ href: "team", text: "Team" }} />;

  return (
    <PageChrome header={header} legacyHeader={legacyHeader}>
      {workspace ? (
        <div className="w-full flex flex-col pt-4">
          <TeamPageClient team={team} />
        </div>
      ) : (
        <div>Workspace not found</div>
      )}
    </PageChrome>
  );
}
