"use client";
import { Navbar } from "@/components/navigation/navbar";
import { Cube } from "@unkey/icons";
import { CreateProjectDialog } from "./_components/create-project/create-project-dialog";

export function ProjectsListNavigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs
        icon={<Cube iconsize="md-medium" className="text-gray-12" />}
      >
        <Navbar.Breadcrumbs.Link href="/projects" active>
          Projects
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <CreateProjectDialog />
    </Navbar>
  );
}
