"use client";
import { useParams, useSelectedLayoutSegments } from "next/navigation";
import { type PropsWithChildren, useState } from "react";
import { ProjectDataProvider } from "./(overview)/data-provider";
import { DeploymentDetailNav } from "./(overview)/deployments/[deploymentId]/deployment-detail-nav";
import { ProjectLayoutContext } from "./(overview)/layout-provider";
import { ProjectNavigation } from "./(overview)/navigations/project-navigation";
import { PendingRedeployBanner } from "./components/pending-redeploy-banner";

export default function ProjectLayoutWrapper({ children }: PropsWithChildren) {
  return <ProjectLayout>{children}</ProjectLayout>;
}

const ProjectLayout = ({ children }: PropsWithChildren) => {
  return (
    <ProjectDataProvider>
      <ProjectLayoutInner>{children}</ProjectLayoutInner>
    </ProjectDataProvider>
  );
};

const ProjectLayoutInner = ({ children }: PropsWithChildren) => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  // Deployments own their chrome: the list renders a PageHeader and the
  // detail renders its own breadcrumb + tabs, so the legacy ProjectNavigation
  // bar would just stack a duplicate. Other app sections still use it.
  // useSelectedLayoutSegments includes the (overview) route group, so match
  // anywhere in the path rather than the first segment. The detail nav lives
  // here, above the scroll container, so it stays fixed while content scrolls.
  const params = useParams();
  const segments = useSelectedLayoutSegments();
  const isDeploymentsSection = segments.includes("deployments");
  const isDeploymentDetail = isDeploymentsSection && typeof params?.deploymentId === "string";

  return (
    <ProjectLayoutContext.Provider
      value={{
        tableDistanceToTop,
      }}
    >
      <div className="h-full flex flex-col overflow-hidden">
        {!isDeploymentsSection && <ProjectNavigation onMount={setTableDistanceToTop} />}
        {isDeploymentDetail ? (
          // The app shell scrolls its whole content column, which would drag
          // the nav off-screen. Bound this region to the viewport (minus the
          // 52px TopNav) so the deployment content scrolls internally and the
          // breadcrumb + tabs stay fixed.
          <div className="flex h-[calc(100dvh-52px)] flex-col">
            <DeploymentDetailNav />
            <div className="min-h-0 flex-1 overflow-auto">{children}</div>
          </div>
        ) : (
          <div className="flex flex-1 min-h-0">
            <div className="flex-1 overflow-auto">{children}</div>
          </div>
        )}
        <PendingRedeployBanner />
      </div>
    </ProjectLayoutContext.Provider>
  );
};
