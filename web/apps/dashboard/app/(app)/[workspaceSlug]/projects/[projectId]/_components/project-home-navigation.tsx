"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Cube, Plus } from "@unkey/icons";
import Link from "next/link";

type ProjectHomeNavigationProps = {
  projectId: string;
};

export const ProjectHomeNavigation = ({ projectId }: ProjectHomeNavigationProps) => {
  const workspace = useWorkspaceNavigation();
  const workspaceSlug = workspace.slug;

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube iconSize="md-medium" className="text-gray-12" />}>
        <Navbar.Breadcrumbs.Link
          href={routes.projects.list({ workspaceSlug })}
          noop={false}
          active={false}
          isLast={false}
        >
          Projects
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={routes.projects.detail({ workspaceSlug, projectId })}
          noop
          active
          isLast
        >
          Overview
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Link href={routes.projects.apps.new({ workspaceSlug, projectId })}>
          <NavbarActionButton title="Create new app" className="cursor-pointer">
            <Plus />
            Create app
          </NavbarActionButton>
        </Link>
      </Navbar.Actions>
    </Navbar>
  );
};
