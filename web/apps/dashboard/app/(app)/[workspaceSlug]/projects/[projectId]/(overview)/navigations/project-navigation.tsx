"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import {
  ArrowDottedRotateAnticlockwise,
  ChevronExpandY,
  Cube,
  Dots,
  DoubleChevronLeft,
  ListRadio,
  Refresh3,
} from "@unkey/icons";
import { Button, InfoTooltip, Separator } from "@unkey/ui";
import { useRef } from "react";
import { RepoDisplay } from "../../../_components/list/repo-display";
import { DisabledWrapper } from "../../components/disabled-wrapper";
import { useBreadcrumbConfig } from "./use-breadcrumb-config";

const BORDER_OFFSET = 1;
type ProjectNavigationProps = {
  projectId: string;
  onMount: (distanceToTop: number) => void;
  onClick: () => void;
  isDetailsOpen: boolean;
  liveDeploymentId?: string | null;
};

export const ProjectNavigation = ({
  projectId,
  onMount,
  isDetailsOpen,
  liveDeploymentId,
  onClick,
}: ProjectNavigationProps) => {
  const workspace = useWorkspaceNavigation();
  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).select(({ project }) => ({
      id: project.id,
      name: project.name,
    })),
  );

  const activeProject = useLiveQuery((q) =>
    q
      .from({ project: collection.projects })
      .where(({ project }) => eq(project.id, projectId))
      .select(({ project }) => ({
        id: project.id,
        name: project.name,
        repositoryFullName: project.repositoryFullName,
      })),
  ).data.at(0);

  const basePath = `/${workspace.slug}/projects`;
  const breadcrumbs = useBreadcrumbConfig({
    projectId,
    basePath,
    projects: projects.data || [],
    activeProject,
  });

  const isOnDeploymentDetail = Boolean(
    breadcrumbs.find((p) => p.id === "deployment-detail")?.active,
  );

  const anchorRef = useRef<HTMLDivElement | null>(null);

  const handleRef = (node: HTMLDivElement | null) => {
    anchorRef.current = node;
    if (node && onMount) {
      const distanceToTop = node.getBoundingClientRect().bottom + BORDER_OFFSET;
      onMount(distanceToTop);
    }
  };

  const getTooltipContent = () => {
    if (!liveDeploymentId) {
      return "No deployments available. Deploy your project to view details.";
    }
    return isDetailsOpen ? "Hide deployment details" : "Show deployment details";
  };

  if (projects.isLoading) {
    const loadingBreadcrumbs = [
      {
        id: "projects",
        children: "Projects",
        href: basePath,
        isIdentifier: false,
        noop: false,
        active: false,
        isLast: false,
      },
      {
        id: "project",
        children: <div className="h-6 w-24 bg-grayA-3 rounded animate-pulse transition-all" />,
        href: "#",
        isIdentifier: true,
        className: "group max-md:hidden",
        noop: true,
        active: false,
        isLast: false,
      },
      {
        id: "subpage",
        children: <div className="h-6 w-20 bg-grayA-3 rounded animate-pulse transition-all" />,
        href: "#",
        noop: true,
        active: true,
        isLast: true,
      },
    ];

    return (
      <Navbar>
        <Navbar.Breadcrumbs icon={<Cube />}>
          {loadingBreadcrumbs.map((crumb) => (
            <Navbar.Breadcrumbs.Link
              key={crumb.id}
              href={crumb.href}
              isIdentifier={crumb.isIdentifier}
              className={crumb.className}
              noop={crumb.noop}
              active={crumb.active}
              isLast={crumb.isLast}
            >
              {crumb.children}
            </Navbar.Breadcrumbs.Link>
          ))}
        </Navbar.Breadcrumbs>
      </Navbar>
    );
  }

  //TODO: Add a proper view here
  if (!activeProject) {
    return <div className="h-full w-full flex items-center justify-center">Project not found</div>;
  }

  return (
    <Navbar ref={handleRef} className="h-[65px]">
      <Navbar.Breadcrumbs icon={<Cube />}>
        {breadcrumbs.map((breadcrumb) => (
          <Navbar.Breadcrumbs.Link
            key={breadcrumb.id}
            href={breadcrumb.href}
            isIdentifier={breadcrumb.isIdentifier}
            active={breadcrumb.active}
            isLast={breadcrumb.isLast}
            noop={breadcrumb.noop}
            className={breadcrumb.className}
          >
            {breadcrumb.quickNavConfig ? (
              <QuickNavPopover
                items={breadcrumb.quickNavConfig.items}
                shortcutKey={breadcrumb.quickNavConfig.shortcutKey}
                activeItemId={breadcrumb.quickNavConfig.activeItemId}
              >
                <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
                  {breadcrumb.children}
                  <ChevronExpandY className="size-4" />
                </div>
              </QuickNavPopover>
            ) : (
              breadcrumb.children
            )}
          </Navbar.Breadcrumbs.Link>
        ))}
      </Navbar.Breadcrumbs>
      <div className="flex gap-4 items-center">
        {activeProject.repositoryFullName && (
          <>
            <div className="text-gray-11 text-xs flex items-center gap-2.5">
              <Refresh3 className="text-gray-12" iconSize="sm-regular" />
              <span>Auto-deploys from pushes to </span>
              <RepoDisplay
                url={`https://github.com/${activeProject.repositoryFullName}`}
                className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] max-w-[130px]"
              />
            </div>
            <Separator orientation="vertical" className="h-5 mx-2 bg-grayA-5" />
          </>
        )}
        <DisabledWrapper tooltipContent="Actions coming soon">
          <div className="gap-2.5 items-center flex">
            <NavbarActionButton title="Visit Project URL">Visit Project URL</NavbarActionButton>
            <Button className="size-7" variant="outline">
              <ListRadio iconSize="sm-regular" />
            </Button>
            <Button className="size-7" variant="outline">
              <ArrowDottedRotateAnticlockwise iconSize="sm-regular" />
            </Button>
            <Button className="size-7" variant="outline">
              <Dots iconSize="sm-regular" />
            </Button>
          </div>
        </DisabledWrapper>
        {!isOnDeploymentDetail && (
          <InfoTooltip
            asChild
            content={getTooltipContent()}
            position={{
              side: "bottom",
              align: "end",
            }}
          >
            <Button
              variant="outline"
              className="size-7"
              disabled={!liveDeploymentId}
              onClick={onClick}
            >
              <DoubleChevronLeft iconSize="lg-medium" className="text-gray-13" />
            </Button>
          </InfoTooltip>
        )}
      </div>
    </Navbar>
  );
};
