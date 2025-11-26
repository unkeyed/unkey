"use client";
import { PageLoading } from "@/components/dashboard/page-loading";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Button, Empty, Input, SettingCard } from "@unkey/ui";
import Link from "next/link";
import { WorkspaceNavbar } from "../workspace-navbar";
import { Client } from "./client";
import { Shell } from "./components/shell";

export default function BillingPage() {
  const { workspace, isLoading: isWorkspaceLoading } = useWorkspace();

  // Derive isLegacy from workspace data
  const isLegacy = workspace?.subscriptions && Object.keys(workspace.subscriptions).length > 0;

  const {
    data: usage,
    isLoading: usageLoading,
    isError,
    error,
  } = trpc.billing.queryUsage.useQuery(undefined, {
    // Only enable query when workspace is loaded AND it's a legacy subscription
    enabled: Boolean(workspace && isLegacy),
    // Skip batching to prevent analytics slowdown from blocking core UI
    trpc: {
      context: {
        skipBatch: true,
      },
    },
    retry: 1,
  });

  // Derive loading state: loading if workspace is loading OR (if legacy, usage is loading)
  const isLoading = isWorkspaceLoading || !workspace || (isLegacy && usageLoading);

  // Wait for workspace to load before proceeding
  if (isLoading) {
    return <PageLoading message="Loading billing..." />;
  }

  if (isError) {
    return (
      <Empty>
        <Empty.Title>Failed to load usage data</Empty.Title>
        <Empty.Description>
          {error?.message ||
            "There was an error loading your usage information. Please try again later."}
        </Empty.Description>
      </Empty>
    );
  }
  if (isLegacy) {
    // Fetch usage data for legacy display
    const verifications = usage?.billableVerifications || 0;
    const ratelimits = usage?.billableRatelimits || 0;

    return (
      <Shell>
        <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
        <div className="w-full">
          <SettingCard
            title="Verifications"
            description="Valid key verifications this month."
            border="top"
          >
            <div className="w-full">
              <Input value={formatNumber(verifications)} />
            </div>
          </SettingCard>
          <SettingCard
            title="Ratelimits"
            description="Valid ratelimits this month."
            border="bottom"
          >
            <div className="w-full">
              <span className="text-xs text-gray-11">
                <Input value={formatNumber(ratelimits)} />
              </span>
            </div>
          </SettingCard>
        </div>

        <SettingCard
          title="Legacy plan"
          border="both"
          description={
            <>
              <p>
                You are on the legacy usage-based plan. You can stay on this plan if you want but
                it's likely more expensive than our new{" "}
                <Link href="https://unkey.com/pricing" className="underline" target="_blank">
                  tiered pricing
                </Link>
                .
              </p>
              <p>If you want to switch over, just let us know.</p>
            </>
          }
        >
          <div className="flex justify-end w-full">
            <Button variant="primary" size="lg">
              <Link href="mailto:support@unkey.dev">Contact us</Link>
            </Button>
          </div>
        </SettingCard>
      </Shell>
    );
  }

  // For non-legacy workspaces, use the Client component with live data
  return <Client />;
}
