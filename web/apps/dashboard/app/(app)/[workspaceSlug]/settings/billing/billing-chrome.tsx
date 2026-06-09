"use client";

import { PageChrome } from "@/components/page-header/page-chrome";
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

/**
 * Shared chrome for the billing screens. Billing renders several bodies
 * (loading, error, legacy, paid/free), so each wraps in this rather than
 * repeating the header.
 */
export function BillingChrome({ children }: { children: ReactNode }) {
  return (
    <PageChrome
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
    </PageChrome>
  );
}
