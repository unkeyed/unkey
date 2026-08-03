import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { env, workosAuthEnv } from "@/lib/env";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";

export const revalidate = 0;

export default async function SettingTeamPage() {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: { quotas: true },
  });

  const team = workspace?.quotas?.team ?? false;
  let teamContent: React.ReactNode = <div>Workspace not found</div>;

  if (workspace) {
    if (env().AUTH_PROVIDER === "local") {
      const { TeamPageClient } = await import("./client");
      teamContent = (
        <div className="flex w-full flex-col">
          <TeamPageClient team={team} />
        </div>
      );
    } else {
      workosAuthEnv();
      const { ManagedTeam } = await import("./managed-team");
      teamContent = <ManagedTeam team={team} />;
    }
  }

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Team</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>{teamContent}</PageBody>
    </PageContainer>
  );
}
