"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { ArrowDottedRotateAnticlockwise, ChevronExpandY, Cube } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";
import { useRef, useState } from "react";
import { useProjectData } from "../data-provider";
import { useBreadcrumbConfig } from "./use-breadcrumb-config";

const CreateDeploymentButton = dynamic(
  () => import("./create-deployment-button").then((m) => m.CreateDeploymentButton),
  { ssr: false },
);

const RedeployDialog = dynamic(
  () =>
    import("../deployments/components/table/components/actions/redeploy-dialog").then(
      (m) => m.RedeployDialog,
    ),
  { ssr: false },
);

const BORDER_OFFSET = 1;
type ProjectNavigationProps = {
  onMount: (distanceToTop: number) => void;
};

export const ProjectNavigation = ({ onMount }: ProjectNavigationProps) => {
  const workspace = useWorkspaceNavigation();
  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).select(({ project }) => ({
      id: project.id,
      name: project.name,
    })),
  );

  const { projectId, project, getDeploymentById } = useProjectData();
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

  const params = useParams();
  const routeDeploymentId = typeof params?.deploymentId === "string" ? params.deploymentId : null;
  const currentDeploymentId = project?.currentDeploymentId;

  const [isRedeployOpen, setIsRedeployOpen] = useState(false);
  const targetDeploymentId = routeDeploymentId ?? currentDeploymentId ?? null;
  const selectedDeployment = targetDeploymentId ? getDeploymentById(targetDeploymentId) : undefined;

  // Close the redeploy dialog when the target deployment changes (e.g. navigation)
  const prevDeploymentIdRef = useRef(targetDeploymentId);
  if (prevDeploymentIdRef.current !== targetDeploymentId) {
    prevDeploymentIdRef.current = targetDeploymentId;
    if (isRedeployOpen) {
      setIsRedeployOpen(false);
    }
  }

  const anchorRef = useRef<HTMLDivElement | null>(null);

  const handleRef = (node: HTMLDivElement | null) => {
    anchorRef.current = node;
    if (node && onMount) {
      const distanceToTop = node.getBoundingClientRect().bottom + BORDER_OFFSET;
      onMount(distanceToTop);
    }
  };

  const redeployTooltip = routeDeploymentId
    ? "Redeploy this deployment"
    : "Redeploy the current active deployment";

  const renderActions = () => {
    return (
      <div className="flex gap-4 items-center">
        <div className="gap-2.5 items-center flex">
          {activeProject?.repositoryFullName && (
            <InfoTooltip
              asChild
              content="Create a deployment from a commit or branch"
              position={{ side: "bottom", align: "end" }}
            >
              <CreateDeploymentButton />
            </InfoTooltip>
          )}
          <InfoTooltip
            asChild
            content={redeployTooltip}
            position={{ side: "bottom", align: "end" }}
          >
            <Button
              className="size-7"
              variant="outline"
              disabled={!selectedDeployment}
              onClick={() => setIsRedeployOpen(true)}
            >
              <ArrowDottedRotateAnticlockwise iconSize="sm-regular" />
            </Button>
          </InfoTooltip>
          {selectedDeployment && (
            <RedeployDialog
              isOpen={isRedeployOpen}
              onClose={() => setIsRedeployOpen(false)}
              selectedDeployment={selectedDeployment}
            />
          )}
        </div>
      </div>
    );
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
      <Navbar.Actions>{renderActions()}</Navbar.Actions>
    </Navbar>
  );
};
