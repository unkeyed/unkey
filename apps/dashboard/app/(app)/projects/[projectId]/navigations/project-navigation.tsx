"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { ArrowDottedRotateAnticlockwise, Cube, Dots, ListRadio, Refresh3 } from "@unkey/icons";
import { Button, Separator } from "@unkey/ui";
import { RepoDisplay } from "../../_components/list/repo-display";

type ProjectNavigationProps = {
  projectId: string;
};

export const ProjectNavigation = ({ projectId }: ProjectNavigationProps) => {
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

  if (projects.isLoading) {
    return (
      <Navbar>
        <Navbar.Breadcrumbs icon={<Cube />}>
          <Navbar.Breadcrumbs.Link href="/projects">Projects</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="#" isIdentifier className="group max-md:hidden" noop>
            <div className="h-6 w-24 bg-grayA-3 rounded animate-pulse transition-all" />
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
    );
  }

  if (!activeProject) {
    return <div className="h-full w-full flex items-center justify-center">Project not found</div>;
  }

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube />}>
        <Navbar.Breadcrumbs.Link href="/projects">Projects</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/projects/${activeProject.id}`}
          isIdentifier
          isLast
          active
          className="flex"
          noop
        >
          <QuickNavPopover
            items={projects.data.map((project) => ({
              id: project.id,
              label: project.name,
              href: `/projects/${project.id}`,
            }))}
            shortcutKey="N"
          >
            <div className="truncate max-w-[120px] h-full">{activeProject.name}</div>
          </QuickNavPopover>
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <div className="flex gap-4 items-center">
        {activeProject.gitRepositoryUrl && (
          <>
            <div className="text-gray-11 text-xs flex items-center gap-2.5">
              <Refresh3 className="text-gray-12" size="sm-regular" />
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
            <ListRadio size="sm-regular" />
          </Button>
          <Button className="size-7" variant="outline">
            <ArrowDottedRotateAnticlockwise size="sm-regular" />
          </Button>
          <Button className="size-7" variant="outline">
            <Dots size="sm-regular" />
          </Button>
        </div>
      </div>
    </Navbar>
  );
};
