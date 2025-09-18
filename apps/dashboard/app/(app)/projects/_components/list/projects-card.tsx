import { CodeBranch, Cube, User } from "@unkey/icons";
import { InfoTooltip, Loading, TimestampInfo } from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";
import { useCallback, useState } from "react";
import { RegionBadges } from "./region-badges";

type ProjectCardProps = {
  name: string;
  domain: string;
  commitTitle: string;
  commitTimestamp?: number | null;
  branch: string;
  author: string;
  regions: string[];
  repository?: string;
  actions: ReactNode;
  projectId: string;
};

export const ProjectCard = ({
  name,
  domain,
  commitTitle,
  commitTimestamp,
  branch,
  author,
  regions,
  repository,
  actions,
  projectId,
}: ProjectCardProps) => {
  const [isNavigating, setIsNavigating] = useState(false);

  const handleLinkClick = useCallback(() => {
    setIsNavigating(true);
  }, []);

  return (
    <div className="relative p-5 flex flex-col border border-grayA-4 hover:border-grayA-7 rounded-2xl w-full gap-5 group transition-all duration-300 [&_a]:z-10 [&_button]:z-10">
      {/* Invisible base clickable layer - covers entire card */}
      <Link
        href={`/projects/${projectId}`}
        className="absolute inset-0 z-0"
        aria-label={`View ${name} project`}
        onClick={handleLinkClick}
      />
      {/*Top Section*/}
      <div className="flex gap-4 items-center">
        <div className="size-10 bg-grayA-3 rounded-[10px] flex items-center justify-center shrink-0 shadow-sm shadow-grayA-8/20">
          {isNavigating ? (
            <Loading size={20} className="text-grayA-11" />
          ) : (
            <Cube size="xl-medium" className="shrink-0 size-5" />
          )}
        </div>
        <div className="flex flex-col w-full gap-2 py-[5px] min-w-0">
          {/*Top Section > Project Name*/}
          <InfoTooltip content={name} asChild position={{ align: "start", side: "top" }}>
            <Link
              href={`/projects/${projectId}`}
              className="font-medium text-sm leading-[14px] text-accent-12 truncate hover:underline"
            >
              {name}
            </Link>
          </InfoTooltip>
          {/*Top Section > Domains/Hostnames*/}
          <InfoTooltip content={domain} asChild position={{ align: "start", side: "top" }}>
            <a
              href={`https://${domain}`}
              target="_blank"
              rel="noopener noreferrer"
              className="relative font-medium text-xs leading-[12px] text-gray-11 truncate max-w-[150px] hover:text-accent-12 transition-colors hover:underline"
            >
              {domain}
            </a>
          </InfoTooltip>
        </div>
        {/*Top Section > Project actions*/}
        <div className="relative">{actions}</div>
      </div>
      {/*Middle Section > Last commit title*/}
      <div className="flex flex-col gap-2">
        <InfoTooltip content={commitTitle} asChild position={{ align: "start", side: "top" }}>
          <Link
            href="#"
            className="text-[13px] font-medium text-accent-12 leading-5 min-w-0 truncate cursor-pointer hover:underline"
          >
            {commitTitle}
          </Link>
        </InfoTooltip>
        <div className="flex gap-2 items-center min-w-0">
          {commitTimestamp ? (
            <TimestampInfo value={commitTimestamp} className="hover:underline whitespace-pre" />
          ) : (
            <span className="text-xs text-gray-12 truncate max-w-[70px]">No deployments</span>
          )}
          <CodeBranch className="text-gray-12 shrink-0" size="sm-regular" />
          <InfoTooltip content={branch} asChild position={{ align: "start", side: "top" }}>
            <span className="text-xs text-gray-12 truncate max-w-[70px]">{branch}</span>
          </InfoTooltip>
          <span className="text-xs text-gray-10">by</span>
          <div className="border border-grayA-6 items-center justify-center rounded-full size-[18px] flex">
            <User className="text-gray-11 shrink-0" size="sm-regular" />
          </div>
          <InfoTooltip content={author} asChild position={{ align: "start", side: "top" }}>
            <span className="text-xs text-gray-12 font-medium truncate max-w-[90px]">{author}</span>
          </InfoTooltip>
        </div>
      </div>
      {/*Bottom Section > Regions*/}
      <RegionBadges regions={regions} repository={repository} />
    </div>
  );
};
