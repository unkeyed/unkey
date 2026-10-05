"use client";
import { PageLoading } from "@/components/dashboard/page-loading";
import { useWorkspaceUsage } from "@/hooks/use-workspace-usage";
import { useFlag } from "@/lib/flags/provider";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { formatNumber } from "@/lib/fmt";
import { useWorkspace } from "@/providers/workspace-provider";
import { Button, Input, SettingCard } from "@unkey/ui";
import Link from "next/link";
import { BillingContainer } from "./billing-container";
import { Client } from "./client";
import { DeployBillingClient } from "./deploy-billing-client";
import { DeployBillingClientV2 } from "./deploy-billing-client-v2";

export default function BillingPage() {
  const { workspace, isLoading: isWorkspaceLoading } = useWorkspace();
  const deployBillingEnabled = useFlag("deployBilling");
  const billingUpgrades = useBillingUIUpgrades();

  // Derive isLegacy from workspace data
  const isLegacy = workspace?.subscriptions && Object.keys(workspace.subscriptions).length > 0;

  const {
    data: usage,
    isLoading: usageLoading,
    isError,
  } = useWorkspaceUsage("current", {
    // Only enable query when workspace is loaded AND it's a legacy subscription
    enabled: Boolean(workspace && isLegacy),
  });

  // Derive loading state: loading if workspace is loading OR (if legacy, usage is loading)
  const isLoading = isWorkspaceLoading || !workspace || (isLegacy && usageLoading);

  // Wait for workspace to load before proceeding
  if (isLoading) {
    return (
      <BillingContainer>
        <PageLoading message="Loading billing..." />
      </BillingContainer>
    );
  }

  if (isLegacy) {
    // Fetch usage data for legacy display
    const usageValue = (value: number | undefined) =>
      isError ? "Unavailable" : formatNumber(value ?? 0);

    return (
      <BillingContainer>
        <div className="flex w-full flex-col items-center gap-6">
          <div className="w-full">
            <SettingCard
              title="Verifications"
              description="Valid key verifications this month."
              border="top"
            >
              <div className="w-full">
                <Input value={usageValue(usage?.api.verifications)} />
              </div>
            </SettingCard>
            <SettingCard
              title="Ratelimits"
              description="Valid ratelimits this month."
              border="bottom"
            >
              <div className="w-full">
                <span className="text-xs text-gray-11">
                  <Input value={usageValue(usage?.api.ratelimits)} />
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
                <Link href="mailto:support@unkey.com">Contact us</Link>
              </Button>
            </div>
          </SettingCard>
        </div>
      </BillingContainer>
    );
  }

  // For non-legacy workspaces, use the Client component with live data. When the
  // deployBilling flag is on, use the separate two-product (API + Compute) page.
  if (billingUpgrades) {
    return <DeployBillingClientV2 />;
  }
  return deployBillingEnabled ? <DeployBillingClient /> : <Client />;
}
