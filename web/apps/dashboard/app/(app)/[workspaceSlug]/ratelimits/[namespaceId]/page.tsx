"use client";
import {
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { use } from "react";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { LogsClient } from "./_overview/logs-client";
import { OverrideIdentifierAction } from "./override-identifier-action";

export default function RatelimitNamespacePage(props: {
  params: Promise<{ namespaceId: string }>;
  searchParams: Promise<{
    identifier?: string;
  }>;
}) {
  const params = use(props.params);
  const { namespaceId } = params;

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Requests</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <OverrideIdentifierAction namespaceId={namespaceId} />
          <CopyableIDButton value={namespaceId} />
        </PageHeaderActions>
      </PageHeader>
      <LogsClient namespaceId={namespaceId} />
    </PageContainer>
  );
}
