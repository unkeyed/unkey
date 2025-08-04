"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Cube } from "@unkey/icons";
import { CreateProjectButton } from "./create-project-button";

export function ProjectsNavigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube size="md-medium" className="text-gray-12" />}>
        <Navbar.Breadcrumbs.Link href="/projects" active className="font-medium">
          Projects
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <CreateProjectButton />
      </Navbar.Actions>
    </Navbar>
  );
}
