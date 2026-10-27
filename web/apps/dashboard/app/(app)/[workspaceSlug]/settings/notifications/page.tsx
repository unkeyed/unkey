"use client";

import { LeaveGuard } from "@/components/leave-guard";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import {
  Button,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  SettingsGroups,
} from "@unkey/ui";
import Link from "next/link";
import { useState } from "react";
import { ConnectedChannels } from "./connected-channels";
import { useDestinationEditor, useRuleEditor } from "./editors";
import { EventRouting } from "./event-routing";
import { NotificationProvider } from "./notification-context";
import { RoutingPanel } from "./routing-panel";

export default function NotificationsPage() {
  return (
    <NotificationProvider>
      <NotificationsPageContent />
    </NotificationProvider>
  );
}

function NotificationsPageContent() {
  const workspace = useWorkspaceNavigation();
  const ruleEditor = useRuleEditor();
  const destinationEditor = useDestinationEditor();
  const [isRoutingOpen, setIsRoutingOpen] = useState(false);

  return (
    <LeaveGuard fallbackHref={routes.settings.notifications({ workspaceSlug: workspace.slug })}>
      <PageContainer>
        <PageHeader className="max-w-[920px] px-6">
          <PageHeaderContent>
            <PageHeaderTitle>Notifications</PageHeaderTitle>
            <PageHeaderDescription>
              Choose who gets notified about events in {workspace.name}.
            </PageHeaderDescription>
          </PageHeaderContent>
          <PageHeaderActions>
            <Button
              variant="outline"
              size="sm"
              render={
                <Link href={routes.account.notifications({ workspaceSlug: workspace.slug })} />
              }
            >
              My preferences
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="sm" />}>
                More
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setIsRoutingOpen(true)}>
                  Preview recipients
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </PageHeaderActions>
        </PageHeader>
        <SettingsGroups className="gap-8">
          <EventRouting onEditRule={ruleEditor.open} />
          <ConnectedChannels onEditDestination={destinationEditor.open} />
        </SettingsGroups>

        {ruleEditor.element}
        {destinationEditor.element}
        <RoutingPanel isOpen={isRoutingOpen} onClose={() => setIsRoutingOpen(false)} />
      </PageContainer>
    </LeaveGuard>
  );
}
