"use client";
import { ProgressCircle } from "@/app/(app)/settings/billing/components/usage";
import { trpc } from "@/lib/trpc/client";
import type { Quotas } from "@unkey/db";
import { Button } from "@unkey/ui";
import Link from "next/link";
import type React from "react";
import { FlatNavItem } from "./app-sidebar/components/nav-items/flat-nav-item";

type Props = {
  quotas: Quotas;
};

export const UsageBanner: React.FC<Props> = ({ quotas }) => {
  const usage = trpc.billing.queryUsage.useQuery(undefined, {
    refetchOnMount: true,
    refetchInterval: 60 * 1000,
  });

  const current = usage.data?.billableTotal ?? 0;
  const max = quotas.requestsPerMonth;

  const shouldUpgrade = current / max > 0.9;

  return (
    <FlatNavItem
      item={{
        tooltip: "Usage",
        icon: () => (
          <ProgressCircle
            value={current}
            max={max}
            color={
              shouldUpgrade
                ? "#DD4527" // error-9
                : "#0A9B8B" // success-9
            }
          />
        ),
        href: "/settings/billing",
        label: `Usage ${Math.round((current / max) * 100).toLocaleString()}%`,
        tag: shouldUpgrade ? (
          <Link href="/settings/billing">
            <Button variant="primary" size="sm">
              Upgrade
            </Button>
          </Link>
        ) : null,
      }}
    />
  );
};
