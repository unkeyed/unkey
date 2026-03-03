import { CodeBranch, Cube, Dots, Earth, Github, User } from "@unkey/icons";
import { Button } from "@unkey/ui";

export const ProjectCardSkeleton = () => {
  return (
    <div className="p-5 flex flex-col border border-grayA-4 hover:border-grayA-7 cursor-pointer rounded-2xl w-full gap-5 group transition-all duration-400">
      {/* Top Section */}
      <div className="flex gap-4 items-center">
        <div className="relative size-10 bg-linear-to-br from-grayA-2 to-grayA-7 rounded-[10px] flex items-center justify-center shrink-0 shadow-xs shadow-grayA-8/20">
          <div className="absolute inset-0 bg-linear-to-br from-grayA-2 to-grayA-4 rounded-[10px] opacity-0 group-hover:opacity-100 transition-opacity duration-500 ease-out" />
          <Cube iconSize="xl-medium" className="relative text-gray-11 opacity-30 shrink-0 size-5" />
        </div>
        <div className="flex flex-col w-full gap-2 py-[5px] min-w-0">
          {/* Project Name Skeleton */}
          <div className="h-[14px] leading-[14px] w-24 bg-grayA-3 rounded-sm animate-pulse" />
          {/* Domain Skeleton */}
          <div className="h-3 leading-3 w-32 bg-grayA-3 rounded-sm animate-pulse" />
        </div>
        {/* Actions Button Skeleton */}
        <Button variant="ghost" size="icon" className="shrink-0" title="Project actions">
          <Dots iconSize="sm-regular" className="text-gray-11 opacity-30 shrink-0" />
        </Button>
      </div>

      {/* Middle Section - Commit Info */}
      <div className="flex flex-col gap-2">
        {/* Commit Title Skeleton */}
        <div className="h-5 leading-5 w-40 bg-grayA-3 rounded-sm animate-pulse" />

        <div className="flex gap-2 items-center min-w-0">
          {/* Commit Date Skeleton */}
          <div className="h-4 w-10 bg-grayA-3 rounded-sm animate-pulse" />
          <span className="text-xs text-gray-11 opacity-30 h-3 flex items-center">on</span>

          {/* Branch Icon */}
          <CodeBranch className="text-gray-12 opacity-30 shrink-0" iconSize="sm-regular" />

          {/* Branch Name Skeleton */}
          <div className="h-4 w-10 max-w-[70px] bg-grayA-3 rounded-sm animate-pulse" />

          <span className="text-xs text-gray-10 opacity-30 h-3 flex items-center">by</span>

          {/* User Avatar */}
          <div className="border border-grayA-6 items-center justify-center rounded-full size-[18px] flex shrink-0">
            <User className="text-gray-11 opacity-30 shrink-0" iconSize="sm-regular" />
          </div>

          {/* Author Name Skeleton */}
          <div className="h-4 w-16 max-w-[90px] bg-grayA-3 rounded-sm animate-pulse" />
        </div>
      </div>

      {/* Bottom Section - Regions Skeleton */}
      <div className="flex gap-2 items-center">
        {/* First region badge skeleton */}
        <div className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] flex items-center gap-1.5 opacity-50">
          <Earth iconSize="lg-medium" className="shrink-0 opacity-30" />
          <div className="h-3 w-16 bg-grayA-6 rounded-sm animate-pulse" />
        </div>

        {/* "+X more" badge skeleton */}
        <div className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] flex items-center opacity-50">
          <div className="h-3 w-12 bg-grayA-6 rounded-sm animate-pulse" />
        </div>

        {/* Repository badge skeleton */}
        <div className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] flex items-center gap-1.5 max-w-[130px] opacity-50">
          <Github iconSize="lg-medium" className="shrink-0 opacity-30" />
          <div className="h-3 w-20 bg-grayA-6 rounded-sm animate-pulse" />
        </div>
      </div>
    </div>
  );
};
