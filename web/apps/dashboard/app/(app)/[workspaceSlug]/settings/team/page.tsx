import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { TeamPageClient } from "./client";

export const revalidate = 0;

export default async function SettingTeamPage() {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: { quotas: true },
  });

  const team = workspace?.quotas?.team ?? false;

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Team</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        {workspace ? (
          <div className="w-full flex flex-col pt-4">
            <TeamPageClient team={team} />
          </div>
        ) : (
          <div>Workspace not found</div>
        )}
      </PageBody>
    </PageContainer>
  );
}
