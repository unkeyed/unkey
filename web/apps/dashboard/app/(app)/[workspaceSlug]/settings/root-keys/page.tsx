"use client";
import { PageChrome } from "@/components/page-header/page-chrome";
import { PageHeader, PageHeaderActions, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { RootKeysListControlCloud } from "./components/control-cloud";
import { RootKeysListControls } from "./components/controls";
import { CreateRootKeyButton } from "./components/dialog/create-rootkey-button";
import { RootKeysList } from "./components/table/root-keys-list";

export default function RootKeysPage() {
  return (
    <PageChrome
      width="full"
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
