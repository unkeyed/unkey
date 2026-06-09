"use client";

import { PageContainer } from "@/components/page-header/page-container";
import {
  Button,
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
    <PageContainer
      header={
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>Billing</PageHeaderTitle>
            <PageHeaderDescription>
              Manage your subscription, usage, and payment methods.
            </PageHeaderDescription>
          </PageHeaderContent>
          <PageHeaderActions>
            <Button asChild variant="outline">
              <Link
                href="https://cal.com/james-r-perkins/sales"
                target="_blank"
                rel="noopener noreferrer"
              >
                Schedule a call
              </Link>
            </Button>
            <Button asChild variant="primary">
              <Link href="mailto:support@unkey.com">Contact us</Link>
            </Button>
          </PageHeaderActions>
        </PageHeader>
      }
    >
      {children}
    </PageContainer>
  );
}
