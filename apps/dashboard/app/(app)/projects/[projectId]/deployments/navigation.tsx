"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { trpc } from "@/lib/trpc/client";
import { Cube, Refresh3 } from "@unkey/icons";
import { RepoDisplay } from "../../_components/list/repo-display";

type DeploymentsNavigationProps = {
  projectId: string;
};

export const DeploymentsNavigation = ({ projectId }: DeploymentsNavigationProps) => {
  const { data: projectData, isLoading } = trpc.deploy.project.list.useInfiniteQuery(
    {}, // No filters needed
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    },
  );

  const projects = projectData?.pages.flatMap((page) => page.projects) ?? [];
  const activeProject = projects.find((p) => p.id === projectId);

  if (isLoading) {
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
    throw new Error(`Project with id "${projectId}" not found`);
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
            items={projects.map((project) => ({
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
      {activeProject.gitRepositoryUrl && (
        <div className="text-gray-11 text-xs flex items-center gap-2.5">
          <Refresh3 className="text-gray-12" size="sm-regular" />
          <span>Auto-deploys from pushes to </span>
          <RepoDisplay
            url={activeProject.gitRepositoryUrl}
            className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] max-w-[130px]"
          />
        </div>
      )}
    </Navbar>
  );
};
