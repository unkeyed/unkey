"use client";
import { DeployFeedbackButton } from "@/components/dashboard/deploy-feedback-button";
import { QuickNavPopover } from "@/components/navbar-popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { ChevronExpandY, Cube, Plus } from "@unkey/icons";

type ProjectHomeNavigationProps = {
  projectId: string;
  projectName: string | undefined;
  onCreateApp: () => void;
};

export const ProjectHomeNavigation = ({
  projectId,
  projectName,
  onCreateApp,
}: ProjectHomeNavigationProps) => {
  const workspace = useWorkspaceNavigation();
  const basePath = `/${workspace.slug}/projects`;

  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).select(({ project }) => ({
      id: project.id,
      name: project.name,
    })),
  );

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube iconSize="md-medium" className="text-gray-12" />}>
        <Navbar.Breadcrumbs.Link href={basePath} noop={false} active={false} isLast={false}>
          Projects
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`${basePath}/${projectId}`}
          noop
          active
          isLast
          className="flex"
        >
          <QuickNavPopover
            items={(projects.data ?? []).map((project) => ({
              id: project.id,
              label: project.name,
              href: `${basePath}/${project.id}`,
            }))}
            shortcutKey="N"
          >
            <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
              {projectName ?? projectId}
              <ChevronExpandY className="size-4" />
            </div>
          </QuickNavPopover>
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <NavbarActionButton title="Create new app" onClick={onCreateApp}>
          <Plus />
          Create app
        </NavbarActionButton>
        <DeployFeedbackButton />
      </Navbar.Actions>
    </Navbar>
  );
};
