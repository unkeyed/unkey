"use client";
import { RootKeysListControlCloud } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/control-cloud";
import { RootKeysListControls } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/controls";
import { CreateRootKeyButton } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/dialog/create-rootkey-button";
import { RootKeysList } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/table/root-keys-list";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { notFound } from "next/navigation";

export default function RootKeysPage() {
  const { user } = useWorkspace();
  if (user && user.role !== "admin") {
    notFound();
  }

  return (
    <PageContainer width="full" data-docs-target="root-key-list">
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
