"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { useParams, usePathname } from "next/navigation";
import { ProjectContentWrapper } from "../../../../components/project-content-wrapper";
import { DeploymentsListControls } from "../../../deployments/components/controls";
import { DeploymentsCardList } from "../../../deployments/components/deployments-card-list";
import { DeploymentsHeader } from "../../../deployments/components/deployments-header";

/**
 * App → Deployments landing. Sits inside the v2b 3-level nav (workspace
 * / project / app); the outer app-leaf sidebar is already contributed
 * by the variant. This page adds a middle "Manage" sidebar with a
 * single active item ("Deployments") and renders the existing list
 * composition in the main pane.
 *
 * The Manage sidebar is scoped to /deployments only — other app
 * sections stay single-column until they earn their own sub-nav.
 */
export default function AppDeploymentsPage() {
  return (
    <div className="flex h-full min-h-0 w-full">
      <ManageSidebar />
      <div className="flex-1 overflow-auto">
        <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
          <DeploymentsHeader />
          <DeploymentsListControls />
          <DeploymentsCardList />
        </ProjectContentWrapper>
      </div>
    </div>
  );
}

function ManageSidebar() {
  const workspace = useWorkspaceNavigation();
  const params = useParams();
  const pathname = usePathname() ?? "";
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";

  const deploymentsHref = `/${workspace.slug}/projects/${projectId}/apps/${appSlug}/deployments`;
  const atDeployments = pathname.endsWith("/deployments");

  return (
    <aside className="flex h-full w-56 shrink-0 flex-col border-r border-grayA-4 py-4">
      <div className="px-4 pb-2 text-[11px] font-medium uppercase tracking-wide text-gray-11">
        Manage
      </div>
      <nav className="flex flex-col">
        <Link
          href={deploymentsHref}
          className={cn(
            "flex items-center gap-2 mx-2 rounded-md px-3 py-1.5 text-[13px] font-medium transition-colors",
            atDeployments
              ? "bg-gray-3 text-accent-12"
              : "text-gray-11 hover:bg-grayA-2 hover:text-accent-12",
          )}
        >
          Deployments
        </Link>
      </nav>
    </aside>
  );
}
