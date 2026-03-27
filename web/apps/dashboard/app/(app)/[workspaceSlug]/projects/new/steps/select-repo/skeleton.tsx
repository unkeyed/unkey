import { CircleDotted } from "@unkey/icons";

export const RepoListItemSkeleton = () => (
  <div className="flex px-4 py-5 items-center h-20">
    <div className="size-[26px] grid place-content-center p-[7px] rounded-lg ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none mr-11">
      <CircleDotted iconSize="sm-medium" className="text-gray-9 opacity-30" />
    </div>
    <div className="flex flex-col gap-1 w-[160px]">
      <div className="h-4 w-24 bg-grayA-3 rounded animate-pulse" />
      <div className="h-3 w-16 bg-grayA-3 rounded animate-pulse" />
    </div>
    <div className="flex gap-2 items-center ml-auto">
      <div className="h-4 w-28 bg-grayA-3 rounded animate-pulse" />
    </div>
    <div className="flex gap-2 items-center">
      <div className="ml-6 w-[140px]">
        <div className="h-7 w-full bg-grayA-3 rounded-lg animate-pulse" />
      </div>
      <div className="h-7 w-[62px] bg-grayA-3 rounded-lg animate-pulse" />
    </div>
  </div>
);

export const SelectRepoSkeleton = () => (
  <div>
    <div className="flex gap-2 w-full pt-1">
      <div className="w-[200px] h-9 bg-grayA-3 rounded-lg animate-pulse shrink-0" />
      <div className="flex-1 h-9 bg-grayA-3 rounded-lg animate-pulse" />
    </div>
    <ul className="mt-3 flex flex-col border rounded-[14px] border-grayA-5 divide-y divide-grayA-5 min-w-[640px] max-h-[462px] overflow-y-auto">
      {Array.from({ length: 3 }).map((_, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: static skeleton list
        <li key={i}>
          <RepoListItemSkeleton />
        </li>
      ))}
    </ul>
  </div>
);
