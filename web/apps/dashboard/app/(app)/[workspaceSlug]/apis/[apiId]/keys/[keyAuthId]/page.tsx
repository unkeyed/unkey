"use client";
import {
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { use } from "react";
import { CreateKeyAction } from "../../create-key-action";
import { KeysClient } from "./_components/keys-client";

export default function APIKeysPage(props: {
  params: Promise<{
    apiId: string;
    keyAuthId: string;
  }>;
}) {
  const params = use(props.params);
  const apiId = params.apiId;
  const keyspaceId = params.keyAuthId;

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Keys</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateKeyAction apiId={apiId} />
        </PageHeaderActions>
      </PageHeader>
      <KeysClient apiId={apiId} keyspaceId={keyspaceId} />
    </PageContainer>
  );
}
