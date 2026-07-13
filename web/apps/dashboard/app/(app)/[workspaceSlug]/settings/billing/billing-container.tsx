"use client";

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
import Link from "next/link";
import type { ReactNode } from "react";

export function BillingContainer({ children }: { children: ReactNode }) {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Billing</PageHeaderTitle>
          <PageHeaderDescription>
            Manage your subscription, usage, and payment methods.
          </PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button
            variant="outline"
            render={
              <Link
                href="https://cal.com/james-r-perkins/sales"
                target="_blank"
                rel="noopener noreferrer"
              />
            }
          >
            Schedule a call
          </Button>
          <Button variant="primary" render={<Link href="mailto:support@unkey.com" />}>
            Contact us
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>{children}</PageBody>
    </PageContainer>
  );
}
