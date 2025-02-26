"use client";
import { ProgressBar } from "@unkey/icons";
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
      <div className="h-[120px]">{chart}</div>
      <Link href={linkPath}>
        <div className="p-6 border-t border-gray-6 flex flex-col gap-1">
          <div className="flex justify-between items-center">
            <div className="flex flex-col">
              <div className="flex gap-3 items-center">
                {icon}
                <div className="text-accent-12 font-semibold">{name}</div>
              </div>
              {secondaryId && <div className="text-accent-11 text-xxs">{secondaryId}</div>}
            </div>
            {rightContent}
          </div>
          <div className="flex items-center w-full justify-between gap-4">{stats}</div>
        </div>
      </Link>
    </div>
  );
};
