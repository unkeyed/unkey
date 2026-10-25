"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  toast,
} from "@unkey/ui";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useState } from "react";
import { DestinationsTab } from "./destinations-tab";
import { useDestinationEditor, useRuleEditor } from "./editors";
import { EmailTab } from "./email-tab";
import { PreviewBadge, PreviewBanner } from "./notification-ui";
import { NotificationPreviewProvider, useNotificationPreview } from "./preview-context";
import { RoutingPanel } from "./routing-panel";
import { RulesTab } from "./rules-tab";

const TABS = ["rules", "destinations", "email"] as const;

export default function NotificationsPage() {
  return (
    <NotificationPreviewProvider>
      <NotificationsPageContent />
    </NotificationPreviewProvider>
  );
}

function NotificationsPageContent() {
  const { actions } = useNotificationPreview();
  const [tab, setTab] = useQueryState(
    "tab",
    parseAsStringLiteral(TABS).withDefault("rules").withOptions({ clearOnDefault: true }),
  );
  const ruleEditor = useRuleEditor();
  const destinationEditor = useDestinationEditor();
  const [isRoutingOpen, setIsRoutingOpen] = useState(false);

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <div className="flex items-center gap-2">
            <PageHeaderTitle>Notifications</PageHeaderTitle>
            <PreviewBadge />
          </div>
        </PageHeaderContent>
      </PageHeader>
      <PageBody className="gap-6">
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
            <TabsTrigger value="email">Your email</TabsTrigger>
          </TabsList>
          <TabsContent value="rules">
            <RulesTab onEditRule={ruleEditor.open} onTestRouting={() => setIsRoutingOpen(true)} />
          </TabsContent>
          <TabsContent value="destinations">
            <DestinationsTab onEditDestination={destinationEditor.open} />
          </TabsContent>
          <TabsContent value="email">
            <EmailTab />
          </TabsContent>
        </Tabs>
      </PageBody>

      {ruleEditor.element}
      {destinationEditor.element}
      <RoutingPanel isOpen={isRoutingOpen} onClose={() => setIsRoutingOpen(false)} />
    </PageContainer>
  );
}
