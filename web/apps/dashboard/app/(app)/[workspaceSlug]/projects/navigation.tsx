"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Cube, Plus } from "@unkey/icons";
import Link from "next/link";

export function ProjectsListNavigation() {
  const workspace = useWorkspaceNavigation();
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube iconSize="md-medium" className="text-gray-12" />}>
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/projects`} active>
          Projects
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Link href={`/${workspace.slug}/projects/new`}>
          <NavbarActionButton title="Create new project" className="cursor-pointer">
            <Plus />
            Create new project
          </NavbarActionButton>
        </Link>
      </Navbar.Actions>
    </Navbar>
  );
}
