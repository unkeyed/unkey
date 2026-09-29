import { IconCircleDottedOutline12 } from "@unkey/icons";
import { Skeleton } from "@unkey/ui";

export const RepoListItemSkeleton = () => (
  <div className="flex px-4 py-5 items-center h-20">
    <div className="size-[26px] grid place-content-center p-[7px] rounded-lg border shadow-sm shadow-grayA-8/20 dark:shadow-none mr-11">
      <IconCircleDottedOutline12 className="text-gray-9 opacity-30" />
    </div>
    <div className="flex flex-col gap-1 w-[160px]">
      <Skeleton className="h-4 w-24 rounded" />
      <Skeleton className="h-3 w-16 rounded" />
    </div>
    <div className="flex gap-2 items-center ml-auto">
      <Skeleton className="h-4 w-28 rounded" />
    </div>
    <div className="flex gap-2 items-center">
      <div className="ml-6 w-[200px]">
        <Skeleton className="h-7 w-full rounded-lg" />
      </div>
      <Skeleton className="h-7 w-[62px] rounded-lg" />
    </div>
  </div>
);

export const SelectRepoSkeleton = () => (
  <div>
    <div className="flex gap-2 w-full pt-1">
      <Skeleton className="w-[200px] h-9 rounded-lg shrink-0" />
      <Skeleton className="flex-1 h-9 rounded-lg" />
    </div>
    <ul className="mt-3 flex flex-col border rounded-lg bg-raised divide-y divide-grayA-5 min-w-[var(--repo-list-w)] max-h-[462px] overflow-y-auto">
      {Array.from({ length: 3 }).map((_, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: static skeleton list
        <li key={i}>
          <RepoListItemSkeleton />
        </li>
      ))}
    </ul>
  </div>
);
