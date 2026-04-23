"use client";

import { isFakeDeployment } from "@/lib/fake-deployments";
import { cn } from "@/lib/utils";
import { useParams } from "next/navigation";
import { DeploymentLayoutProvider } from "../../../../deployments/[deploymentId]/layout-provider";
import LegacyDeploymentOverview from "../../../../deployments/[deploymentId]/page";
import { FakeDetail } from "./_fake-detail";

/**
 * Per-app deployment detail route.
 *
 * Layout: the outer app-leaf sidebar is contributed by the v2b navbar
 * variant. This page renders a thin tab row + the deployment body. Only
 * the "Overview" tab exists today; the row is kept so adding Logs /
 * Settings later is a 1-line change.
 *
 * Real deployments delegate to the existing legacy detail composition
 * via `DeploymentLayoutProvider` + the legacy `page.tsx` default
 * export, so we inherit build / approval / cancelled / ready / progress
 * views with zero duplication. Mock deployments (id prefix
 * `dep_fake_`) render a smaller `FakeDetail` placeholder because the
 * legacy composition reads from real-data collections.
 */
export default function AppDeploymentDetailPage() {
  const params = useParams();
  const deploymentId = typeof params?.deploymentId === "string" ? params.deploymentId : "";
  const isFake = isFakeDeployment({ id: deploymentId });

  return (
    <div className="flex h-full min-h-0 w-full flex-col">
      <DetailTabs activeTab="overview" />
      <div className="flex-1 overflow-auto">
        {isFake ? (
          <FakeDetail deploymentId={deploymentId} />
        ) : (
          <DeploymentLayoutProvider deploymentId={deploymentId}>
            <LegacyDeploymentOverview />
          </DeploymentLayoutProvider>
        )}
      </div>
    </div>
  );
}

type TabId = "overview";

function DetailTabs({ activeTab }: { activeTab: TabId }) {
  const tabs: Array<{ id: TabId; label: string }> = [{ id: "overview", label: "Overview" }];
  return (
    <div className="border-b border-grayA-4 px-6">
      <nav className="flex items-center gap-1">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            className={cn(
              "relative px-3 py-3 text-[13px] font-medium transition-colors",
              tab.id === activeTab
                ? "text-accent-12 after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-accent-12"
                : "text-gray-11 hover:text-accent-12",
            )}
          >
            {tab.label}
          </button>
        ))}
      </nav>
    </div>
  );
}
