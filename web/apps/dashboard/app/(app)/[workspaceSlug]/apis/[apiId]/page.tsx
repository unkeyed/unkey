"use client";
import { LogsClient } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/logs-client";
import {
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { use } from "react";
import { CreateKeyAction } from "./create-key-action";

export default function ApiPage(props: { params: Promise<{ apiId: string }> }) {
  const params = use(props.params);
  const apiId = params.apiId;

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Requests</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateKeyAction apiId={apiId} />
        </PageHeaderActions>
      </PageHeader>
      <LogsClient apiId={apiId} />
    </PageContainer>
  );
}
