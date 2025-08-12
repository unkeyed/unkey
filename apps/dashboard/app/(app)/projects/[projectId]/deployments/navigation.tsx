"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { Cube } from "@unkey/icons";
import { trpc } from "@/lib/trpc/client";

type DeploymentsNavigationProps = {
  projectId: string;
};

export const DeploymentsNavigation = ({
  projectId,
}: DeploymentsNavigationProps) => {
  const { data: projectData, isLoading } =
    trpc.deploy.project.list.useInfiniteQuery(
      {}, // No filters needed
      {
        getNextPageParam: (lastPage) => lastPage.nextCursor,
        staleTime: Number.POSITIVE_INFINITY,
        refetchOnMount: false,
        refetchOnWindowFocus: false,
      }
    );

  const projects = projectData?.pages.flatMap((page) => page.projects) ?? [];

  const activeProject = projects.find((p) => p.id === projectId);

  if (isLoading) {
    return (
      <Navbar>
        <Navbar.Breadcrumbs icon={<Cube />}>
          <Navbar.Breadcrumbs.Link href="/projects">
            Projects
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href="#"
            isIdentifier
            className="group max-md:hidden"
            noop
          >
            <div className="h-6 w-24 bg-grayA-3 rounded animate-pulse transition-all" />
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
    );
  }

  if (!activeProject) {
    throw new Error(`Project with id "${projectId}" not found`);
  }

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube />}>
        <Navbar.Breadcrumbs.Link href="/projects">
          Projects
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/projects/${activeProject.id}`}
          isIdentifier
          isLast
          active
          className="flex"
          noop
        >
          <QuickNavPopover
            items={projects.map((project) => ({
              id: project.id,
              label: project.name,
              href: `/projects/${project.id}`,
            }))}
            shortcutKey="N"
          >
            <div className="text-accent-10 group-hover:text-accent-12 truncate max-w-[120px] h-full">
              {activeProject.name}
            </div>
          </QuickNavPopover>
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
};
