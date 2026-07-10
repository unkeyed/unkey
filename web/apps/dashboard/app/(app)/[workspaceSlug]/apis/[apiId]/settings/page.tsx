"use client";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
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
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Settings</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <SettingsClient apiId={apiId} />
      </PageBody>
    </PageContainer>
  );
}
