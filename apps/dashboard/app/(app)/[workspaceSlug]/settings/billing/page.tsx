"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { Button, Empty, Input, SettingCard } from "@unkey/ui";
import Link from "next/link";
import { WorkspaceNavbar } from "../workspace-navbar";
import { Client } from "./client";
import { Shell } from "./components/shell";

export default function BillingPage() {
  const workspace = useWorkspaceNavigation();

  // Early return if workspace is not available
  if (!workspace) {
    return (
      <div>
        <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
        <Empty>
          <Empty.Title>Workspace not found</Empty.Title>
          <Empty.Description>Unable to load workspace information.</Empty.Description>
        </Empty>
      </div>
    );
  }

  // Check for legacy subscriptions
  const isLegacy = workspace?.subscriptions && Object.keys(workspace.subscriptions).length > 0;

  if (isLegacy) {
    // Fetch usage data for legacy display
    const { data: usage, isLoading: usageLoading } = trpc.billing.queryUsage.useQuery();
    const verifications = usage?.billableVerifications || 0;
    const ratelimits = usage?.billableRatelimits || 0;

    if (usageLoading) {
      return (
        <div className="animate-pulse">
          <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
          <Shell>
            <div className="w-full h-[500px] bg-gray-100 dark:bg-gray-800 rounded-lg" />
          </Shell>
        </div>
      );
    }

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
