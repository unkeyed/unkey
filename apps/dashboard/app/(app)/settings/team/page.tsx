import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navigation } from "@/components/navigation/navigation";
import { PageContent } from "@/components/page-content";
import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import TeamPageClient from "./team-client";
import { navigation } from "../constants";
import { Gear } from "@unkey/icons";

export const revalidate = 0;

export default async function SettingTeamPage() {
  const orgId = await getOrgId();
  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: { quota: true },
  });

  const team = ws?.quota?.team ?? false;

  return (
    <div>
      <Navigation href="/settings/team" name="Settings" icon={<Gear />} />
      <PageContent>
        <SubMenu navigation={navigation} segment="team" />
        <div className="mb-20 flex flex-col gap-8 mt-8">
          <TeamPageClient team={team} />
        </div>
      </PageContent>
    </div>
  );
}
