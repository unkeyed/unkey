"use client";
import {
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsGroups,
} from "@unkey/ui";
import { use } from "react";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: Promise<{
    apiId: string;
  }>;
};

export default function SettingsPage(props: Props) {
  const params = use(props.params);
  const { apiId } = params;

  return (
    <PageContainer>
      <PageHeader className="max-w-[920px]">
        <PageHeaderContent>
          <PageHeaderTitle>Settings</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <SettingsGroups>
        <SettingsClient apiId={apiId} />
      </SettingsGroups>
    </PageContainer>
  );
}
