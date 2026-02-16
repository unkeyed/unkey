"use client";
import { useSidebar } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";
import { ChevronLeft } from "@unkey/icons";
import Link from "next/link";

type ProjectSidebarHeaderProps = {
  backHref: string;
};

export const ProjectSidebarHeader = ({ backHref }: ProjectSidebarHeaderProps) => {
  const { state } = useSidebar();
  const isCollapsed = state === "collapsed";

  return (
    <div className="flex flex-col gap-1 px-2 mb-2">
      <Link
        href={backHref}
        className={cn(
          "flex items-center gap-1 text-gray-9 hover:text-gray-12 transition-colors text-sm py-1 rounded-md",
          isCollapsed && "justify-center",
        )}
      >
        <ChevronLeft iconSize="xl-medium" />
        <div className="flex items-center gap-2 px-1 text-gray-12 font-medium text-sm truncate">
          <span className="truncate">Projects</span>
        </div>
      </Link>
    </div>
  );
};
