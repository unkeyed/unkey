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
import { KeySettingsDialog } from "../../../_components/key-settings-dialog";
import { KeyDetailsLogsClient } from "./logs-client";

export default function KeyDetailsPage(props: {
  params: Promise<{ apiId: string; keyAuthId: string; keyId: string }>;
}) {
  const params = use(props.params);
  const { apiId, keyAuthId: keyspaceId, keyId } = params;

  const { data, error } = trpc.api.keys.list.useQuery({
    keyAuthId: keyspaceId,
    keyIds: [{ operator: "is", value: keyId }],
    identities: null,
    limit: 1,
    names: null,
  });

  if (error) {
    throw new Error(`Failed to fetch key details: ${error.message}`);
  }

  const key = data?.keys.find((k) => k.id === keyId);
  const title = key?.name || shortenId(keyId);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle className="truncate" title={title}>
            {title}
          </PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          {key ? <KeySettingsDialog keyData={key} apiId={apiId} keyspaceId={keyspaceId} /> : null}
          <CopyableIDButton value={keyId} />
        </PageHeaderActions>
      </PageHeader>
      <KeyDetailsLogsClient apiId={apiId} keyspaceId={keyspaceId} keyId={keyId} />
    </PageContainer>
  );
}
