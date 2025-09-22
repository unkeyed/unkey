"use client";
import { PostAuthInvitationHandler } from "@/components/auth/post-auth-invitation-handler";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Nodes } from "@unkey/icons";
import { Loading } from "@unkey/ui";
import { useSearchParams, redirect } from "next/navigation";
import { ApiListClient } from "./_components/api-list-client";
import { CreateApiButton } from "./_components/create-api-button";

export default function ApisOverviewPage() {
  const workspace = useWorkspaceNavigation();

  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";

  return (
    <div>
      <PostAuthInvitationHandler />
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/apis`} active>
            APIs
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <CreateApiButton
            key="createApi"
            defaultOpen={isNewApi}
            workspaceSlug={workspace.slug}
          />
        </Navbar.Actions>
      </Navbar>
      <ApiListClient workspaceSlug={workspace.slug} />
    </div>
  );
}
