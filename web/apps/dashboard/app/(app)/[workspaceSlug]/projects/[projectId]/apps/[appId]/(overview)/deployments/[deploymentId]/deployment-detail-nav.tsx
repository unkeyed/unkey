"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { shortenId } from "@/lib/shorten-id";
import { ArrowDottedRotateAnticlockwise, Ban, ChevronRight } from "@unkey/icons";
import {
  Button,
  PageHeaderDescription,
  PageHeaderTitle,
  SecondaryNav,
  SecondaryNavGroup,
  SecondaryNavItem,
  SecondaryNavTitle,
  Tabs,
  TabsList,
  TabsTrigger,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useParams, useSelectedLayoutSegments } from "next/navigation";
import { useState } from "react";
import { useProjectData } from "../../data-provider";
import { isCancellableDeploymentStatus } from "../components/table/components/actions/deployment-action-eligibility";

const RedeployDialog = dynamic(
  () =>
    import("../components/table/components/actions/redeploy-dialog").then((m) => m.RedeployDialog),
  { ssr: false },
);

const CancelDialog = dynamic(
  () => import("../components/table/components/actions/cancel-dialog").then((m) => m.CancelDialog),
  { ssr: false },
);

function useParam(name: string): string {
  const params = useParams();
  const value = params?.[name];
  return typeof value === "string" ? value : "";
}

/**
 * Tab definitions for a deployment. Tabs are routes, not in-page state: each
 * href is a route and the active tab is the path segment after the deployment
 * id, derived so it works wherever the nav is mounted.
 */
function useDeploymentTabs() {
  const workspace = useWorkspaceNavigation();
  const projectId = useParam("projectId");
  const appId = useParam("appId");
  const deploymentId = useParam("deploymentId");

  const segments = useSelectedLayoutSegments();
  const tabIndex = segments.indexOf(deploymentId) + 1;
  const activeTab = (tabIndex > 0 && segments[tabIndex]) || "deployment";

  const deploymentsHref = `/${workspace.slug}/projects/${projectId}/apps/${appId}/deployments`;
  const base = `${deploymentsHref}/${deploymentId}`;

  const tabs = [
    { value: "deployment", label: "Deployment", href: base },
    { value: "logs", label: "Logs", href: `${base}/logs` },
    { value: "requests", label: "Requests", href: `${base}/requests` },
  ];

  return { tabs, activeTab, deploymentsHref, deploymentId };
}

/** Horizontal underline tab strip, shared by the breadcrumb and header variants. */
function DeploymentTabsRow() {
  const { tabs, activeTab } = useDeploymentTabs();
  return (
    <div className="border-b border-grayA-4 px-4">
      <Tabs value={activeTab}>
        <TabsList className="-ml-2 h-auto justify-start gap-1 rounded-none bg-transparent p-0">
          {tabs.map((tab) => (
            <TabsTrigger
              key={tab.value}
              value={tab.value}
              asChild
              className="relative rounded-none bg-transparent px-3 py-3 text-[13px] font-medium text-gray-11 shadow-none transition-colors hover:bg-transparent hover:text-accent-12 data-[state=active]:bg-transparent data-[state=active]:text-accent-12 data-[state=active]:shadow-none after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-accent-12 after:opacity-0 data-[state=active]:after:opacity-100"
            >
              <Link href={tab.href}>{tab.label}</Link>
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
    </div>
  );
}

/** `Deployments › <id>` row, shared by both nav variants. */
export function DeploymentBreadcrumb() {
  const { deploymentsHref, deploymentId } = useDeploymentTabs();
  return (
    <div className="flex items-center gap-1 px-4 pt-3">
      <Link
        href={deploymentsHref}
        className="rounded px-1 py-0.5 text-xs font-medium text-gray-11 hover:bg-grayA-3 hover:text-accent-12"
      >
        Deployments
      </Link>
      <ChevronRight className="size-2.5 text-gray-9" />
      <span className="px-1 py-0.5 font-mono text-xs font-medium text-accent-12">
        {shortenId(deploymentId)}
      </span>
    </div>
  );
}

/** Breadcrumb + horizontal underline tabs. */
export function DeploymentDetailNav() {
  return (
    <div className="shrink-0">
      <DeploymentBreadcrumb />
      <DeploymentTabsRow />
    </div>
  );
}

/**
 * PlanetScale-style header: the deployment id as a prominent page title with
 * its actions on the right, then the tabs. No breadcrumb — the persistent app
 * rail is the way back.
 */
export function DeploymentDetailHeaderNav() {
  const { deploymentId } = useDeploymentTabs();
  const { getDeploymentById } = useProjectData();
  const deployment = getDeploymentById(deploymentId);

  const [isRedeployOpen, setIsRedeployOpen] = useState(false);
  const [isCancelOpen, setIsCancelOpen] = useState(false);
  const [cancelled, setCancelled] = useState(false);
  const canCancel = deployment
    ? isCancellableDeploymentStatus(deployment.status) && !cancelled
    : false;

  // The commit message is what humans recognise; the id is an internal handle
  // and still shows in the deployment card below.
  const title = deployment?.gitCommitMessage || shortenId(deploymentId);
  const subtitle = deployment
    ? [deployment.gitBranch, deployment.gitCommitSha?.slice(0, 7)].filter(Boolean).join(" · ")
    : null;

  return (
    <div className="shrink-0">
      <div className="flex items-start justify-between gap-3 px-4 pt-5 pb-3">
        <div className="min-w-0">
          <PageHeaderTitle className="truncate">{title}</PageHeaderTitle>
          {subtitle && (
            <PageHeaderDescription className="font-mono">{subtitle}</PageHeaderDescription>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {canCancel && (
            <Button variant="outline" onClick={() => setIsCancelOpen(true)}>
              <Ban iconSize="sm-regular" />
              Cancel deployment
            </Button>
          )}
          <Button variant="outline" disabled={!deployment} onClick={() => setIsRedeployOpen(true)}>
            <ArrowDottedRotateAnticlockwise iconSize="sm-regular" />
            Redeploy
          </Button>
        </div>
      </div>
      <DeploymentTabsRow />
      {deployment && (
        <RedeployDialog
          isOpen={isRedeployOpen}
          onClose={() => setIsRedeployOpen(false)}
          selectedDeployment={deployment}
        />
      )}
      {deployment && canCancel && (
        <CancelDialog
          isOpen={isCancelOpen}
          onClose={() => setIsCancelOpen(false)}
          onCancelled={() => setCancelled(true)}
          deployment={deployment}
        />
      )}
    </div>
  );
}

/** Vertical SecondaryNav rail variant of the deployment tabs. */
export function DeploymentDetailSidebar() {
  const { tabs, activeTab } = useDeploymentTabs();

  return (
    <SecondaryNav aria-label="Deployment details">
      <SecondaryNavTitle>Deployment details</SecondaryNavTitle>
      <SecondaryNavGroup>
        {tabs.map((tab) => (
          <SecondaryNavItem key={tab.value} asChild active={activeTab === tab.value}>
            <Link href={tab.href}>{tab.label}</Link>
          </SecondaryNavItem>
        ))}
      </SecondaryNavGroup>
    </SecondaryNav>
  );
}
