"use client";
import { RootKeysListControls } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/controls";
import { CreateRootKeyButton } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/dialog/create-rootkey-button";
import { RootKeysList } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/table/root-keys-list";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import { IconPlusOutline18 } from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { notFound } from "next/navigation";

export default function RootKeysPage() {
  const { user } = useWorkspace();
  const workspace = useWorkspaceNavigation();
  const rootKeyBuilder = useFlag("rootKeyBuilder");
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
          {rootKeyBuilder ? (
            <Button
              variant="primary"
              size="sm"
              render={<Link href={routes.settings.rootKeyNew({ workspaceSlug: workspace.slug })} />}
            >
              <IconPlusOutline18 />
              New Root Key
            </Button>
          ) : (
            <CreateRootKeyButton />
          )}
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col">
          <RootKeysListControls />
          <RootKeysList />
        </div>
      </PageBody>
    </PageContainer>
  );
}
