"use client";
import { PostAuthInvitationHandler } from "@/components/auth/post-auth-invitation-handler";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { ApiListClient } from "./_components/api-list-client";
import { CreateApiButton } from "./_components/create-api-button";

export default function ApisOverviewPage() {
  const workspace = useWorkspaceNavigation();

  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";

  return (
    <PageContainer>
      <PostAuthInvitationHandler />
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Keyspaces</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateApiButton key="createApi" defaultOpen={isNewApi} workspaceSlug={workspace.slug} />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <ApiListClient workspaceSlug={workspace.slug} />
      </PageBody>
    </PageContainer>
  );
}
