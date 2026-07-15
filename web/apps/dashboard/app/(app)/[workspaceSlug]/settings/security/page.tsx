"use client";

import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { TwoFactorAuth } from "./two-factor-auth";

export default function SecuritySettingsPage() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Security</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <div className="w-full flex flex-col pt-4">
          <TwoFactorAuth />
        </div>
      </PageBody>
    </PageContainer>
  );
}
