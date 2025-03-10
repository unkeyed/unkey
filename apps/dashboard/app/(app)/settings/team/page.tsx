import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navigation } from "@/components/navigation/navigation";
import { PageContent } from "@/components/page-content";
import { getCurrentUser, getWorkspace } from "@/lib/auth/actions";
import { Gear } from "@unkey/icons";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { navigation } from "../constants";
import { TeamPageClient } from "./team-client";

export const dynamic = "force-dynamic";

export default async function TeamPage() {
  const user = await getCurrentUser();
  if (!user || !user.orgId) {
    return null;
  }

  const { orgId } = user;
  const workspace = await getWorkspace(orgId);

  if (workspace.plan === "free") {
    return (
      <div>
        <Navigation href="/settings/team" name="Settings" icon={<Gear />} />
        <PageContent>
          <SubMenu navigation={navigation} segment="team" />
          <div className="mb-20 flex flex-col gap-8 mt-8">
            <Empty>
              <Empty.Title>This is a personal account</Empty.Title>
              <Empty.Description>You can only manage teams in paid workspaces.</Empty.Description>
              <Empty.Actions>
                <Link href="/new">
                  <Button>Create a new workspace</Button>
                </Link>
              </Empty.Actions>
            </Empty>
          </div>
        </PageContent>
      </div>
    );
  }

  return <TeamPageClient />;
}
