"use client";
import { BuilderAside } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/builder-aside";
import { RootKeysListControls } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/controls";
import { RootKeysList } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/table/root-keys-list";
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
  ResourceList,
  buttonVariants,
} from "@unkey/ui";
import { notFound } from "next/navigation";
import { useState } from "react";

export default function RootKeysPage() {
  const { user } = useWorkspace();
  const [asideOpen, setAsideOpen] = useState(false);

  if (user && user.role !== "admin") {
    notFound();
  }

  return (
    <PageContainer className="flex-1" data-docs-target="root-key-list">
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
          <Button
            variant="primary"
            size="sm"
            className="rounded-md px-3"
            onClick={() => setAsideOpen(true)}
          >
            <IconPlusOutline18 />
            New Root Key
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody className="flex-1">
        <ResourceList>
          <RootKeysListControls />
          <RootKeysList />
        </ResourceList>
      </PageBody>
      <BuilderAside isOpen={asideOpen} onClose={() => setAsideOpen(false)} />
    </PageContainer>
  );
}
