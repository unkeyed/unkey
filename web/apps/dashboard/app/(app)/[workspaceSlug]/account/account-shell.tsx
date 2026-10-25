import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type React from "react";

export function AccountShell({
  children,
  notificationsHref,
}: {
  children: React.ReactNode;
  notificationsHref?: Route;
}) {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Account settings </PageHeaderTitle>
          <PageHeaderDescription>Manage your profile and sign-in security.</PageHeaderDescription>
        </PageHeaderContent>
        {notificationsHref ? (
          <PageHeaderActions>
            <Button variant="outline" render={<Link href={notificationsHref} />}>
              Personal notifications
            </Button>
          </PageHeaderActions>
        ) : null}
      </PageHeader>
      <PageBody>{children}</PageBody>
    </PageContainer>
  );
}
