import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { WorkspaceNavbar } from "../workspace-navbar";
import { TeamPageClient } from "./client";

export const revalidate = 0;

export default async function SettingTeamPage() {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    where: { orgId, deletedAtM: { isNull: true } },
    with: { quota: true },
  });

  const team = workspace?.quota?.team ?? false;

  return workspace ? (
    <>
      <WorkspaceNavbar activePage={{ href: "team", text: "Team" }} />
      <div className="flex flex-col w-full max-w-6xl mx-auto px-6 py-8">
        <TeamPageClient team={team} />
      </div>
    </>
  ) : (
    <div>
      <div>Workspace not found</div>
    </div>
  );
}
