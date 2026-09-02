"use client";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { use } from "react";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: Promise<{
    namespaceId: string;
  }>;
};

export default function SettingsPage(props: Props) {
  const params = use(props.params);
  const namespaceId = params.namespaceId;

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Settings</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CopyableIDButton value={namespaceId} />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <SettingsClient namespaceId={namespaceId} />
      </PageBody>
    </PageContainer>
  );
}
