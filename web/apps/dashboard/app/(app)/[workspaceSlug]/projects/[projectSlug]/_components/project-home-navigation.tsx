"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Cube, Plus } from "@unkey/icons";
import Link from "next/link";

type ProjectHomeNavigationProps = {
  projectSlug: string;
};

export const ProjectHomeNavigation = ({ projectSlug }: ProjectHomeNavigationProps) => {
  const workspace = useWorkspaceNavigation();
  const basePath = `/${workspace.slug}/projects`;

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube iconSize="md-medium" className="text-gray-12" />}>
        <Navbar.Breadcrumbs.Link href={basePath} noop={false} active={false} isLast={false}>
          Projects
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`${basePath}/${projectSlug}`} noop active isLast>
          Overview
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Link href={`${basePath}/${projectSlug}/apps/new`}>
          <NavbarActionButton title="Create new app" className="cursor-pointer">
            <Plus />
            Create app
          </NavbarActionButton>
        </Link>
      </Navbar.Actions>
    </Navbar>
  );
};
