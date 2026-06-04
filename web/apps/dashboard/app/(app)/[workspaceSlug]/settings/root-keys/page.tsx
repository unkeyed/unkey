"use client";
import { PageChrome } from "@/components/page-header/page-chrome";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { PageHeader, PageHeaderActions, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { RootKeysListControlCloud } from "./components/control-cloud";
import { RootKeysListControls } from "./components/controls";
import { CreateRootKeyButton } from "./components/dialog/create-rootkey-button";
import { RootKeysList } from "./components/table/root-keys-list";
import { Navigation } from "./navigation";

export default function RootKeysPage() {
  const workspace = useWorkspaceNavigation();

  return (
    <PageChrome
      width="full"
      legacyHeader={
        <Navigation
          workspace={{
            id: workspace.id,
            name: workspace.name,
            slug: workspace.slug ?? "",
          }}
          activePage={{
            href: "root-keys",
            text: "Root Keys",
          }}
        />
      }
      header={
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>Root Keys</PageHeaderTitle>
          </PageHeaderContent>
          <PageHeaderActions>
            <CreateRootKeyButton />
          </PageHeaderActions>
        </PageHeader>
      }
    >
      <div className="flex flex-col">
        <RootKeysListControls />
        <RootKeysListControlCloud />
        <RootKeysList />
      </div>
    </PageChrome>
  );
}
