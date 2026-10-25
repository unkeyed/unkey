"use client";

import { LeaveGuard } from "@/components/leave-guard";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
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
  toast,
} from "@unkey/ui";
import Link from "next/link";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useState } from "react";
import { DestinationsTab } from "./destinations-tab";
import { useDestinationEditor, useRuleEditor } from "./editors";
import { PreviewBadge, PreviewBanner } from "./notification-ui";
import { NotificationPreviewProvider, useNotificationPreview } from "./preview-context";
import { RoutingPanel } from "./routing-panel";
import { RulesTab } from "./rules-tab";

const TABS = ["rules", "destinations"] as const;

export default function NotificationsPage() {
  return (
    <NotificationPreviewProvider>
      <NotificationsPageContent />
    </NotificationPreviewProvider>
  );
}

function NotificationsPageContent() {
  const { actions } = useNotificationPreview();
  const workspace = useWorkspaceNavigation();
  const [tab, setTab] = useQueryState(
    "tab",
    parseAsStringLiteral(TABS).withDefault("rules").withOptions({ clearOnDefault: true }),
  );
  const ruleEditor = useRuleEditor();
  const destinationEditor = useDestinationEditor();
  const [isRoutingOpen, setIsRoutingOpen] = useState(false);

  return (
    <LeaveGuard fallbackHref={routes.settings.notifications({ workspaceSlug: workspace.slug })}>
      <PageContainer>
        <PageHeader className="max-w-[920px] px-6">
          <PageHeaderContent>
            <div className="flex items-center gap-2">
              <PageHeaderTitle>Notifications</PageHeaderTitle>
              <PreviewBadge />
            </div>
            <PageHeaderDescription>
              Choose who hears about events in your workspace.
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
              Personal notifications
            </Button>
          </PageHeaderActions>
        </PageHeader>
        <SettingsGroups className="gap-5">
          <PreviewBanner
            onReset={() => {
              actions.reset();
              toast.success("Sample data restored");
            }}
          />
          <Tabs
            value={tab}
            onValueChange={(next: string) => {
              const parsed = TABS.find((candidate) => candidate === next);
              if (parsed) {
                setTab(parsed);
              }
            }}
            className="flex flex-col gap-6"
          >
            <TabsList className="w-fit">
              <TabsTrigger value="rules">Rules</TabsTrigger>
              <TabsTrigger value="destinations">Destinations</TabsTrigger>
            </TabsList>
            <TabsContent value="rules">
              <RulesTab onEditRule={ruleEditor.open} onTestRouting={() => setIsRoutingOpen(true)} />
            </TabsContent>
            <TabsContent value="destinations">
              <DestinationsTab onEditDestination={destinationEditor.open} />
            </TabsContent>
          </Tabs>
        </SettingsGroups>

        {ruleEditor.element}
        {destinationEditor.element}
        <RoutingPanel isOpen={isRoutingOpen} onClose={() => setIsRoutingOpen(false)} />
      </PageContainer>
    </LeaveGuard>
  );
}
