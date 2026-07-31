import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import type React from "react";

export function AccountShell({ children }: { children: React.ReactNode }) {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Account</PageHeaderTitle>
          <PageHeaderDescription>Manage your profile and sign-in security.</PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>{children}</PageBody>
    </PageContainer>
  );
}
