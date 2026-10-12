"use client";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import {
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsGroups,
} from "@unkey/ui";
import { use } from "react";
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
      <PageHeader className="max-w-[920px]">
        <PageHeaderContent>
          <PageHeaderTitle>Settings</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CopyableIDButton value={namespaceId} />
        </PageHeaderActions>
      </PageHeader>
      <SettingsGroups>
        <SettingsClient namespaceId={namespaceId} />
      </SettingsGroups>
    </PageContainer>
  );
}
