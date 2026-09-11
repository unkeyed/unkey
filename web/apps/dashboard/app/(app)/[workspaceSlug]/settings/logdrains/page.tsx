"use client";

import { SUPPORT_MAILTO } from "@/lib/support";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useState } from "react";
import { CreateLogdrainButton } from "./create-logdrain-button";
import { CreateLogdrainPanel } from "./create-logdrain-panel";
import { LogdrainsList } from "./logdrains-list";

export default function LogdrainsPage() {
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const { limits, isLoading: isWorkspaceLoading } = useWorkspace();
  const drains = trpc.logdrain.list.useQuery();
  const isLoading = isWorkspaceLoading || drains.isLoading || drains.isError;
  const isAtLimit = (drains.data?.length ?? 0) >= (limits?.logdrainsMax ?? 0);
  const canCreate = !isLoading && !isAtLimit;
  const openCreatePanel = () => {
    if (canCreate) {
      setIsCreateOpen(true);
    }
  };

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Log Drains</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateLogdrainButton onClick={openCreatePanel} disabled={!canCreate} />
        </PageHeaderActions>
      </PageHeader>
      <PageBody className="gap-4">
        {!isLoading && isAtLimit && (
          <AlertBanner variant="warning">
            <AlertBannerTitle>Log drain limit reached</AlertBannerTitle>
            <AlertBannerDescription>
              Contact support to enable log drains or increase this workspace's allowance. Existing
              log drains remain available.
            </AlertBannerDescription>
            <AlertBannerActions>
              <Button variant="outline" size="sm" render={<Link href={SUPPORT_MAILTO} />}>
                Contact support
              </Button>
            </AlertBannerActions>
          </AlertBanner>
        )}
        <LogdrainsList onCreate={openCreatePanel} canCreate={canCreate} />
      </PageBody>

      <CreateLogdrainPanel
        isOpen={isCreateOpen && canCreate}
        onClose={() => setIsCreateOpen(false)}
      />
    </PageContainer>
  );
}
