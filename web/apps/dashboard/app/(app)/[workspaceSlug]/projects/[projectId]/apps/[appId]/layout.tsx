"use client";
import { useParams, useSelectedLayoutSegments } from "next/navigation";
import { type PropsWithChildren, useState } from "react";
import { ProjectDataProvider } from "./(overview)/data-provider";
import {
  DeploymentDetailNav,
  DeploymentDetailSidebar,
} from "./(overview)/deployments/[deploymentId]/deployment-detail-nav";
import { DeploymentNavVariantToggle } from "./(overview)/deployments/[deploymentId]/deployment-nav-variant-toggle";
import { useDeploymentNavVariant } from "./(overview)/deployments/[deploymentId]/use-deployment-nav-variant";
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
  const [navVariant] = useDeploymentNavVariant();

  return (
    <ProjectLayoutContext.Provider
      value={{
        tableDistanceToTop,
      }}
    >
      <div className="h-full flex flex-col overflow-hidden">
        {!isDeploymentsSection && <ProjectNavigation onMount={setTableDistanceToTop} />}
        {/* The app shell scrolls its whole content column, which would drag the
            nav off-screen. Bound this region to the viewport (minus the 52px
            TopNav) so the content scrolls internally and the nav stays fixed. */}
        {isDeploymentDetail ? (
          navVariant === "crumb" ? (
            // Trail lives in the top bar and the deploy-specific rail replaces
            // the global app sidebar, so the content area is chrome-free.
            <div className="flex h-[calc(100dvh-52px)] flex-col">
              <div className="min-h-0 flex-1 overflow-auto">{children}</div>
            </div>
          ) : navVariant === "sidebar" ? (
            // SecondaryNav rail handles wayfinding, so no breadcrumb here.
            // No flex-1 on the row: it would override the explicit height and
            // grow with content, scrolling the whole region (rail included).
            <div className="flex h-[calc(100dvh-52px)]">
              <DeploymentDetailSidebar />
              <div className="min-h-0 flex-1 overflow-auto">{children}</div>
            </div>
          ) : (
            <div className="flex h-[calc(100dvh-52px)] flex-col">
              <DeploymentDetailNav />
              <div className="min-h-0 flex-1 overflow-auto">{children}</div>
            </div>
          )
        ) : (
          <div className="flex flex-1 min-h-0">
            <div className="flex-1 overflow-auto">{children}</div>
          </div>
        )}
        <PendingRedeployBanner />
      </div>
      {isDeploymentsSection && <DeploymentNavVariantToggle />}
    </ProjectLayoutContext.Provider>
  );
};
