"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderBack,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { EmailTab } from "../../settings/notifications/email-tab";
import { AccountUnavailable } from "../account-unavailable";

export function PersonalNotifications() {
  const workspace = useWorkspaceNavigation();
  const { user } = useWorkspace();

  return (
    <PageContainer>
      <PageHeader className="max-w-[920px]">
        <PageHeaderContent>
          <PageHeaderBack
            render={<Link href={routes.account.overview({ workspaceSlug: workspace.slug })} />}
          >
            Account settings
          </PageHeaderBack>
          <PageHeaderTitle>Notifications</PageHeaderTitle>
          <PageHeaderDescription className="break-words">
            Your email preferences for {workspace.name} only.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody className="max-w-[920px] gap-6">
        {user?.impersonator ? (
          <AccountUnavailable reason="impersonation" />
        ) : (
          <EmailTab key={`${workspace.id}:${user?.id}`} />
        )}
      </PageBody>
    </PageContainer>
  );
}
