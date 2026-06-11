"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { shortenId } from "@/lib/shorten-id";
import { ChevronRight } from "@unkey/icons";
import { Tabs, TabsList, TabsTrigger } from "@unkey/ui";
import Link from "next/link";
import { useParams, useSelectedLayoutSegments } from "next/navigation";

function useParam(name: string): string {
  const params = useParams();
  const value = params?.[name];
  return typeof value === "string" ? value : "";
}

/**
 * Breadcrumb + tab strip for a single deployment. The tabs are routes, not
 * in-page state: each trigger is a `Link`, so URLs stay shareable and
 * back/forward work. The active tab is the path segment after the
 * deployment id, derived so the bar works wherever it's mounted.
 */
export function DeploymentDetailNav() {
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
    { value: "resources", label: "Resources", href: `${base}/resources` },
  ];

  return (
    <div className="shrink-0">
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
    </div>
  );
}
