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
import { OverrideIdentifierAction } from "../override-identifier-action";
import { LogsClient } from "./components/logs-client";

export default function RatelimitLogsPage(props: { params: Promise<{ namespaceId: string }> }) {
  const params = use(props.params);
  const { namespaceId } = params;

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Logs</PageHeaderTitle>
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
