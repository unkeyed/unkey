"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { useNavbarVariant } from "@/hooks/use-navbar-variant";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { ChevronExpandY, Cube } from "@unkey/icons";
import { useRef } from "react";
import { useProjectData } from "../data-provider";
import { useBreadcrumbConfig } from "./use-breadcrumb-config";

const BORDER_OFFSET = 1;
type ProjectNavigationProps = {
  onMount: (distanceToTop: number) => void;
  // Retained for caller compatibility (project layout still passes these).
  // The Create Deployment / Redeploy / details-drawer toggle buttons used
  // to live here — they're app-level and will move to the app leaf once
  // applications are a first-class UI concept (see Andreas, 2026-04-21).
  onClick?: () => void;
  isDetailsOpen?: boolean;
  currentDeploymentId?: string | null;
};

export const ProjectNavigation = ({ onMount }: ProjectNavigationProps) => {
  const workspace = useWorkspaceNavigation();
  const { variant } = useNavbarVariant();
  // v2b carries the breadcrumb trail in its global top header, so the
  // in-page project nav becomes redundant there.
  if (variant === "v2b") {
    return null;
  }
  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).select(({ project }) => ({
      id: project.id,
      name: project.name,
    })),
  );

  const { projectId, project } = useProjectData();
  const activeProject = project
    ? { id: project.id, name: project.name, repositoryFullName: project.repositoryFullName }
    : undefined;

  const basePath = `/${workspace.slug}/projects`;
  const breadcrumbs = useBreadcrumbConfig({
    projectId,
    basePath,
    projects: projects.data || [],
    activeProject,
  });

  const anchorRef = useRef<HTMLDivElement | null>(null);

  const handleRef = (node: HTMLDivElement | null) => {
    anchorRef.current = node;
    if (node && onMount) {
      const distanceToTop = node.getBoundingClientRect().bottom + BORDER_OFFSET;
      onMount(distanceToTop);
    }
  };

  if (projects.isLoading) {
    const loadingBreadcrumbs = [
      {
        id: "projects",
        children: "Projects",
        href: basePath,
        noop: false,
        active: false,
        isLast: false,
      },
      {
        id: "project",
        children: <div className="h-6 w-24 bg-grayA-3 rounded-sm animate-pulse transition-all" />,
        href: "#",
        className: "group max-md:hidden",
        noop: true,
        active: false,
        isLast: false,
      },
      {
        id: "subpage",
        children: <div className="h-6 w-20 bg-grayA-3 rounded-sm animate-pulse transition-all" />,
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

  if (!activeProject) {
    return (
      <Navbar>
        <Navbar.Breadcrumbs icon={<Cube />}>
          <Navbar.Breadcrumbs.Link href={basePath} noop={false} active={false} isLast={false}>
            Projects
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
    );
  }

  return (
    <Navbar ref={handleRef} className="h-[65px]">
      <Navbar.Breadcrumbs icon={<Cube />}>
        {breadcrumbs.map((breadcrumb) => (
          <Navbar.Breadcrumbs.Link
            key={breadcrumb.id}
            href={breadcrumb.href}
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
    </Navbar>
  );
};
