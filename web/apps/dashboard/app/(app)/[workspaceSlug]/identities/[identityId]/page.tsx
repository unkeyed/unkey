"use client";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import {
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { use } from "react";
import { IdentitySettingsDialog } from "./components/identity-settings-dialog";
import { IdentityDetailsLogsClient } from "./logs-client";

export default function IdentityDetailsPage(props: {
  params: Promise<{ identityId: string }>;
}) {
  const { identityId } = use(props.params);

  const { data: identity } = trpc.identity.getById.useQuery({ identityId });
  const title = identity?.externalId || shortenId(identityId);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle className="truncate" title={title}>
            {title}
          </PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <IdentitySettingsDialog identityId={identityId} />
          <CopyableIDButton value={identityId} />
        </PageHeaderActions>
      </PageHeader>
      <IdentityDetailsLogsClient identityId={identityId} />
    </PageContainer>
  );
}
