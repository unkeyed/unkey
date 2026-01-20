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
import { useSelectedLayoutSegments } from "next/navigation";
import { useRef } from "react";
import { RepoDisplay } from "../../_components/list/repo-display";

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
        gitRepositoryUrl: project.gitRepositoryUrl,
      })),
  ).data.at(0);

  const basePath = `/${workspace.slug}/projects`;

  const segments = useSelectedLayoutSegments() ?? [];
  const activeSubPage = segments[0]; // undefined, "deployments", "sentinel-logs", or "openapi-diff"

  const subPages = [
    { id: "overview", label: "Overview", href: `${basePath}/${projectId}`, segment: undefined },
    {
      id: "deployments",
      label: "Deployments",
      href: `${basePath}/${projectId}/deployments`,
      segment: "deployments",
    },
    {
      id: "sentinel-logs",
      label: "Sentinel Logs",
      href: `${basePath}/${projectId}/sentinel-logs`,
      segment: "sentinel-logs",
    },
    {
      id: "openapi-diff",
      label: "OpenAPI Diff",
      href: `${basePath}/${projectId}/openapi-diff`,
      segment: "openapi-diff",
    },
  ];

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

  const currentSubPage = subPages.find((page) => page.segment === activeSubPage) || subPages[0];

  if (projects.isLoading) {
    return (
      <Navbar>
        <Navbar.Breadcrumbs icon={<Cube />}>
          <Navbar.Breadcrumbs.Link href={basePath}>Projects</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="#" isIdentifier className="group max-md:hidden" noop>
            <div className="h-6 w-24 bg-grayA-3 rounded animate-pulse transition-all" />
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="#" noop active isLast>
            <div className="h-6 w-20 bg-grayA-3 rounded animate-pulse transition-all" />
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
    );
  }

  if (!activeProject) {
    return <div className="h-full w-full flex items-center justify-center">Project not found</div>;
  }
  return (
    <Navbar ref={handleRef}>
      <Navbar.Breadcrumbs icon={<Cube />}>
        <Navbar.Breadcrumbs.Link href={basePath}>Projects</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`${basePath}/${activeProject.id}`}
          isIdentifier
          className="flex"
          noop
        >
          <QuickNavPopover
            items={projects.data.map((project) => ({
              id: project.id,
              label: project.name,
              href: `${basePath}/${project.id}`,
            }))}
            shortcutKey="N"
          >
            <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
              {activeProject.name}
              <ChevronExpandY className="size-4" />
            </div>
          </QuickNavPopover>
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={currentSubPage.href} noop active isLast>
          <QuickNavPopover
            items={subPages.map((page) => ({
              id: page.id,
              label: page.label,
              href: page.href,
            }))}
            shortcutKey="M"
          >
            <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
              {currentSubPage.label}
              <ChevronExpandY className="size-4" />
            </div>
          </QuickNavPopover>
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <div className="flex gap-4 items-center">
        {activeProject.gitRepositoryUrl && (
          <>
            <div className="text-gray-11 text-xs flex items-center gap-2.5">
              <Refresh3 className="text-gray-12" iconSize="sm-regular" />
              <span>Auto-deploys from pushes to </span>
              <RepoDisplay
                url={activeProject.gitRepositoryUrl}
                className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] max-w-[130px]"
              />
            </div>
            <Separator orientation="vertical" className="h-5 mx-2 bg-grayA-5" />
          </>
        )}
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
          <InfoTooltip
            asChild
            content={getTooltipContent()}
            position={{
              side: "bottom",
              align: "end",
            }}
          >
            <Button
              variant="ghost"
              className="size-7"
              disabled={!liveDeploymentId}
              onClick={onClick}
            >
              <DoubleChevronLeft iconSize="lg-medium" className="text-gray-13" />
            </Button>
          </InfoTooltip>
        </div>
      </div>
    </Navbar>
  );
};
