"use client";
import { ProgressBar } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";

export type StatsCardProps = {
  name: string;
  secondaryId?: string;
  linkPath: string;
  chart: ReactNode;
  stats: ReactNode;
  rightContent?: ReactNode;
  icon?: ReactNode;
};

export const StatsCard = ({
  name,
  secondaryId,
  linkPath,
  chart,
  stats,
  rightContent,
  icon = <ProgressBar className="text-accent-11" />,
}: StatsCardProps) => {
  return (
    <div className="flex flex-col border border-gray-6 rounded-xl overflow-hidden">
      <div className="h-[140px]">{chart}</div>
      <Link href={linkPath}>
        <div className="p-4 md:p-6 border-t border-gray-6 flex flex-col gap-2">
          <div className="flex justify-between items-center">
            <div className="flex flex-col flex-grow min-w-0">
              <div className="flex gap-2 md:gap-3 items-center">
                <span className="flex-shrink-0">{icon}</span>
                <Tooltip>
                  <TooltipTrigger>
                    <div className="text-accent-12 font-semibold truncate w-[220px] md:w-[280px] text-left">
                      {name}
                    </div>
                  </TooltipTrigger>
                  <TooltipContent className="bg-gray-12 text-gray-1 px-3 py-2 border border-accent-6 shadow-md font-medium text-xs">
                    {name}
                  </TooltipContent>
                </Tooltip>
              </div>
              {secondaryId && (
                <Tooltip>
                  <TooltipTrigger>
                    <div className="text-left text-accent-11 text-xxs overflow-hidden text-ellipsis w-[240px] md:w-[300px]">
                      {secondaryId}
                    </div>
                  </TooltipTrigger>
                  <TooltipContent className="bg-gray-12 text-gray-1 px-3 py-2 border border-accent-6 shadow-md font-medium text-xs">
                    {secondaryId}
                  </TooltipContent>
                </Tooltip>
              )}
            </div>
            {rightContent && <div className="flex-shrink-0">{rightContent}</div>}
          </div>
          <div className="flex items-center w-full justify-between gap-3 md:gap-4 mt-2">
            {stats}
          </div>
        </div>
      </Link>
    </div>
  );
};
