"use client";

import { CodeBranch, Cube, Dots, Earth, Github, User } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { LogsProvider } from "../logs/context/logs";
import { LogsControlCloud } from "./_components/control-cloud";
import { LogsControls } from "./_components/controls";
import { ProjectsNavigation } from "./_components/navigation";

const projects = [
  {
    name: "dashboard",
    domain: "api.gateway.com",
    commitTitle: "feat: add paginated tRPC endpoint for projects (#3697)",
    commitDate: "Jul 24",
    branch: "main",
    author: "Oz",
    regions: [
      "us-east-1",
      "us-west-1",
      "us-west-2",
      "eu-west-1",
      "eu-central-1",
      "ap-southeast-1",
      "ap-northeast-1",
      "ca-central-1",
    ],
    repository: "unkeyed/unkey",
  },
  {
    name: "unkey-marketing-super-long-name-testing-truncate",
    domain: "api.gateway-long-name-truncate-works-i-guess-slug-dk.com",
    commitTitle: "feat: add paginated tRPC endpoint for projects (#3697)",
    commitDate: "Jul 24",
    branch: "main-branch-but-super-long-redundant-weird-shit",
    author: "Oz-but-his-name-is-very-very-long",
    regions: [
      "us-east-1",
      "us-west-1",
      "us-west-2",
      "eu-west-1",
      "eu-central-1",
      "ap-southeast-1",
      "ap-northeast-1",
      "ca-central-1",
    ],
    repository: "unkeyed/unkey-super-weird-branc-name-long",
  },
  {
    name: "unkey-works",
    domain: "api.gateway.com",
    commitTitle: "feat: add paginated tRPC endpoint for projects (#3697)",
    commitDate: "Jul 24",
    branch: "main",
    author: "Oz",
    regions: [
      "us-east-1",
      "us-west-1",
      "us-west-2",
      "eu-west-1",
      "eu-central-1",
      "ap-southeast-1",
      "ap-northeast-1",
      "ca-central-1",
    ],
    repository: "unkeyed/unkey",
  },
];

export function ProjectsClient() {
  return (
    <div>
      <ProjectsNavigation />
      <LogsProvider>
        <LogsControls />
        <LogsControlCloud />
      </LogsProvider>
      {/*Container*/}
      <div className="p-4">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {projects.map((project) => (
            <ProjectCard key={project.name} {...project} />
          ))}
        </div>
      </div>
    </div>
  );
}
type RegionBadgesProps = {
  regions: string[];
  repository?: string;
};

const RegionBadges = ({ regions, repository }: RegionBadgesProps) => {
  const visibleRegions = regions.slice(0, 1);
  const remainingRegions = regions.slice(1);
  const remainingCount = remainingRegions.length;

  return (
    <div className="flex gap-2 items-center">
      {visibleRegions.map((region) => (
        <div
          key={region}
          className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] flex items-center gap-1.5"
        >
          <Earth size="lg-medium" className="shrink-0" />
          {region}
        </div>
      ))}
      {remainingCount > 0 && (
        <InfoTooltip
          content={
            <div className="space-y-1">
              {remainingRegions.map((region) => (
                <div key={region} className="text-xs font-medium flex items-center gap-1.5">
                  <div className="w-1 h-1 bg-gray-8 rounded-full" />
                  {region}
                </div>
              ))}
            </div>
          }
        >
          <div className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] flex items-center">
            +{remainingCount} more
          </div>
        </InfoTooltip>
      )}
      {repository && (
        <InfoTooltip content={repository} asChild>
          <div className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] flex items-center gap-1.5 max-w-[130px]">
            <Github size="lg-medium" className="shrink-0" />
            <span className="truncate">{repository}</span>
          </div>
        </InfoTooltip>
      )}
    </div>
  );
};

type ProjectCardProps = {
  name: string;
  domain: string;
  commitTitle: string;
  commitDate: string;
  branch: string;
  author: string;
  regions: string[];
  repository?: string;
};

const ProjectCard = ({
  name,
  domain,
  commitTitle,
  commitDate,
  branch,
  author,
  regions,
  repository,
}: ProjectCardProps) => (
  <div className="p-5 flex flex-col border border-grayA-4 hover:border-grayA-7 cursor-pointer rounded-2xl w-full gap-5 group transition-all duration-200">
    {/*Top Section*/}
    <div className="flex gap-4 items-center">
      <div className="relative size-10 bg-gradient-to-br from-grayA-2 to-grayA-7 rounded-[10px] flex items-center justify-center shrink-0 shadow-sm shadow-grayA-8/20">
        <div className="absolute inset-0 bg-gradient-to-br from-grayA-2 to-grayA-4 rounded-[10px] opacity-0 group-hover:opacity-100 transition-opacity duration-500 ease-out" />
        <Cube size="xl-medium" className="relative text-gray-11 shrink-0 size-5" />
      </div>
      <div className="flex flex-col w-full gap-2 py-[5px] min-w-0">
        {/*Top Section > Project Name*/}
        <InfoTooltip content={name} asChild>
          <div className="font-medium text-sm leading-[14px] text-accent-12 truncate">{name}</div>
        </InfoTooltip>
        {/*Top Section > Domains/Hostnames*/}
        <InfoTooltip content={domain} asChild>
          <a
            href={`https://${domain}`}
            target="_blank"
            rel="noopener noreferrer"
            className="font-medium text-xs leading-[12px] text-gray-11 truncate hover:text-accent-12 transition-colors hover:underline"
          >
            {domain}
          </a>
        </InfoTooltip>
      </div>
      {/*Top Section > Project actions*/}
      <Button variant="ghost" size="icon" className="mb-auto shrink-0" title="Project actions">
        <Dots size="sm-regular" />
      </Button>
    </div>
    {/*Middle Section > Last commit title*/}
    <div className="flex flex-col gap-2">
      <InfoTooltip content={commitTitle} asChild>
        <div className="text-[13px] font-medium text-accent-12 leading-5 min-w-0 truncate cursor-pointer">
          {commitTitle}
        </div>
      </InfoTooltip>
      <div className="flex gap-2 items-center min-w-0">
        <span className="text-xs text-gray-11">{commitDate} on</span>
        <CodeBranch className="text-gray-12 shrink-0" size="sm-regular" />
        <InfoTooltip content={branch} asChild>
          <span className="text-xs text-gray-12 truncate max-w-[70px]">{branch}</span>
        </InfoTooltip>
        <span className="text-xs text-gray-10">by</span>
        <div className="border border-grayA-6 items-center justify-center rounded-full size-[18px] flex">
          <User className="text-gray-11 shrink-0" size="sm-regular" />
        </div>
        <InfoTooltip content={author} asChild>
          <span className="text-xs text-gray-12 font-medium truncate max-w-[90px]">{author}</span>
        </InfoTooltip>
      </div>
    </div>
    {/*Bottom Section > Regions*/}
    <RegionBadges regions={regions} repository={repository} />
  </div>
);
