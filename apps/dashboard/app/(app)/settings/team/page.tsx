import { getCurrentUser, getWorkspace } from "@/lib/auth/actions";
import { TeamPageClient } from "./team-client";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { navigation } from "../constants";
import { Gear } from "@unkey/icons";
import Link from "next/link";

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
        <Navbar>
          <Navbar.Breadcrumbs icon={<Gear />}>
            <Navbar.Breadcrumbs.Link href="/settings/team" active>
              Settings
            </Navbar.Breadcrumbs.Link>
          </Navbar.Breadcrumbs>
        </Navbar>
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

  return <TeamPageClient initialData={{ orgId, workspace }} />;
}