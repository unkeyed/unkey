"use client";
import { RootKeysListControlCloud } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/control-cloud";
import { RootKeysListControls } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/controls";
import { CreateRootKeyButton } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/dialog/create-rootkey-button";
import { RootKeysList } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/table/root-keys-list";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";

// Root Keys' top-level home. The component tree still lives under
// settings/root-keys; the old settings URL redirects here.
export default function RootKeysPage() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Root Keys</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateRootKeyButton />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col">
          <RootKeysListControls />
          <RootKeysListControlCloud />
          <RootKeysList />
        </div>
      </PageBody>
    </PageContainer>
  );
}
