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
import { PreviewBadge, PreviewBanner } from "../../settings/notifications/notification-ui";
import { NotificationPreviewProvider } from "../../settings/notifications/preview-context";
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
          <div className="flex items-center gap-2">
            <PageHeaderTitle>Personal notifications</PageHeaderTitle>
            <PreviewBadge />
          </div>
          <PageHeaderDescription>
            Email preferences for {workspace.name}. These settings apply only to you in this
            workspace.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody className="max-w-[920px] gap-6">
        <PreviewBanner />
        {user?.impersonator ? (
          <AccountUnavailable reason="impersonation" />
        ) : (
          <NotificationPreviewProvider>
            <EmailTab />
          </NotificationPreviewProvider>
        )}
      </PageBody>
    </PageContainer>
  );
}
