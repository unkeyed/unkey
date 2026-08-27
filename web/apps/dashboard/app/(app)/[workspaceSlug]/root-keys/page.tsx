"use client";
import { BuilderAside } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/builder-aside";
import { RootKeysListControls } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/controls";
import { CreateRootKeyButton } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/dialog/create-rootkey-button";
import { RootKeysListBuilder } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/table/root-keys-list-builder";
import { RootKeysListLegacy } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/table/root-keys-list-legacy";
import { useFlag } from "@/lib/flags/provider";
import { useWorkspace } from "@/providers/workspace-provider";
import { IconBookBookmarkOutline18, IconPlusOutline18 } from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  buttonVariants,
} from "@unkey/ui";
import { notFound } from "next/navigation";
import { useState } from "react";

export default function RootKeysPage() {
  const { user } = useWorkspace();
  const rootKeyBuilder = useFlag("rootKeyBuilder");
  const [asideOpen, setAsideOpen] = useState(false);

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
          <a
            href="https://www.unkey.com/docs/security/overview#root-keys"
            target="_blank"
            rel="noopener noreferrer"
            className={buttonVariants({ variant: "outline", size: "sm", className: "px-3" })}
          >
            <IconBookBookmarkOutline18 />
            Documentation
          </a>
          {rootKeyBuilder ? (
            <Button
              variant="primary"
              size="sm"
              className="rounded-md px-3"
              onClick={() => setAsideOpen(true)}
            >
              <IconPlusOutline18 />
              New Root Key
            </Button>
          ) : (
            <CreateRootKeyButton />
          )}
        </PageHeaderActions>
      </PageHeader>
      <PageBody className="gap-3 pt-3 pb-0">
        <RootKeysListControls />
        {rootKeyBuilder ? <RootKeysListBuilder /> : <RootKeysListLegacy />}
      </PageBody>
      {rootKeyBuilder ? (
        <BuilderAside isOpen={asideOpen} onClose={() => setAsideOpen(false)} />
      ) : null}
    </PageContainer>
  );
}
