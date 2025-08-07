"use client";
import { Navbar } from "@/components/navigation/navbar";
import { Cube } from "@unkey/icons";
import { CreateProjectDialog } from "./_components/create-project/create-project-dialog";

export function ProjectsListNavigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube size="md-medium" className="text-gray-12" />}>
        <Navbar.Breadcrumbs.Link href="/projects" active className="font-medium">
          Projects
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <CreateProjectDialog />
    </Navbar>
  );
}
