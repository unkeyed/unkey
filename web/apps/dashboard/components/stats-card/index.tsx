"use client";
import { ProgressBar } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
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
      <Link href={linkPath} prefetch>
        <div className="p-4 md:p-6 border-t border-gray-6 flex flex-col gap-2">
          <div className="flex justify-between items-center">
            <div className="flex flex-col grow min-w-0">
              <div className="flex gap-2 md:gap-3 items-center">
                <span className="shrink-0">{icon}</span>
                <InfoTooltip variant="inverted" position={{ side: "top" }} content={name}>
                  <div className="text-accent-12 font-semibold truncate w-[220px] md:w-[280px] text-left">
                    {name}
                  </div>
                </InfoTooltip>
              </div>
              {secondaryId && (
                <InfoTooltip variant="inverted" position={{ side: "top" }} content={secondaryId}>
                  <div className="text-left text-accent-11 text-xxs overflow-hidden text-ellipsis w-[240px] md:w-[300px]">
                    {secondaryId}
                  </div>
                </InfoTooltip>
              )}
            </div>
            {rightContent && <div className="shrink-0">{rightContent}</div>}
          </div>
          <div className="flex items-center w-full justify-between gap-3 md:gap-4 mt-2">
            {stats}
          </div>
        </div>
      </Link>
    </div>
  );
};
