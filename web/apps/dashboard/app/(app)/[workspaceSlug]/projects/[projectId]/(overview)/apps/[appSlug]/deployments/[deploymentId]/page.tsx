"use client";

import { useV2bDeploymentsVariant } from "@/hooks/use-v2b-deployments-variant";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { isFakeDeployment } from "@/lib/fake-deployments";
import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import { ChevronRight } from "@unkey/icons";
import Link from "next/link";
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
  const { variant: subVariant } = useV2bDeploymentsVariant();
  // `c` promotes the deployment crumb up into V2BTopHeader; `d` swaps it
  // for an eyebrow header; `e` (drawer) and `f` (split) provide their own
  // chrome via the layout. Only `a` and `b` keep the original in-page row.
  const headerKind: "crumb" | "eyebrow" | "none" =
    subVariant === "a" || subVariant === "b"
      ? "crumb"
      : subVariant === "d"
        ? "eyebrow"
        : "none";

  return (
    <div className="flex h-full min-h-0 w-full flex-col">
      {headerKind === "crumb" ? <DetailBreadcrumb deploymentId={deploymentId} /> : null}
      {headerKind === "eyebrow" ? <EyebrowHeader deploymentId={deploymentId} /> : null}
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

function DetailBreadcrumb({ deploymentId }: { deploymentId: string }) {
  const workspace = useWorkspaceNavigation();
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";
  const listHref = `/${workspace.slug}/projects/${projectId}/apps/${appSlug}/deployments`;

  return (
    <div className="flex shrink-0 items-center gap-1 px-4 pt-3">
      <Link
        href={listHref}
        className="rounded px-1 py-0.5 text-xs font-medium text-gray-11 hover:bg-grayA-3 hover:text-accent-12"
      >
        Deployments
      </Link>
      <ChevronRight className="size-2.5 text-gray-9" iconSize="sm-thin" />
      <span className="px-1 py-0.5 font-mono text-xs font-medium text-accent-12">
        {shortenId(deploymentId)}
      </span>
    </div>
  );
}

/**
 * Sub-variant `d`: a slim back-link kicker above a page H1, in lieu of
 * a breadcrumb chain. The workspace/project/app context still lives in
 * V2BTopHeader, so one back-action is enough wayfinding.
 */
function EyebrowHeader({ deploymentId }: { deploymentId: string }) {
  const workspace = useWorkspaceNavigation();
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";
  const listHref = `/${workspace.slug}/projects/${projectId}/apps/${appSlug}/deployments`;

  return (
    <div className="flex shrink-0 flex-col gap-1 border-b border-grayA-4 px-6 py-4">
      <Link
        href={listHref}
        className="inline-flex w-fit items-center gap-1 text-[12px] font-medium text-gray-11 hover:text-accent-12"
      >
        <ChevronRight className="size-3 rotate-180" iconSize="sm-thin" />
        Deployments
      </Link>
      <h1 className="font-mono text-lg font-semibold text-accent-12">{shortenId(deploymentId)}</h1>
    </div>
  );
}

type TabId = "overview";

function DetailTabs({ activeTab }: { activeTab: TabId }) {
  const tabs: Array<{ id: TabId; label: string }> = [{ id: "overview", label: "Overview" }];
  return (
    <div className="border-b border-grayA-4 px-4">
      <nav className="-ml-2 flex items-center gap-1">
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
