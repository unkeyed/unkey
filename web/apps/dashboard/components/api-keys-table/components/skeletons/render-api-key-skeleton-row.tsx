import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { cn } from "@/lib/utils";
import { IconDotsOutline12 } from "@unkey/icons";
import type { DataTableColumnDef } from "@unkey/ui";
import { KeyColumnSkeleton } from "@unkey/ui";
import { API_KEY_COLUMN_IDS } from "../../columns/create-api-key-columns";
import { UsageColumnSkeleton } from "../skeletons";

const ApiKeyIdColumnSkeleton = () => (
  <div className="flex flex-col items-start px-[18px] py-[6px]">
    <div className="flex gap-4 items-center">
      <Skeleton className="size-5" />
      <div className="flex flex-col gap-1">
        <Skeleton className="h-2 w-40" />
        <Skeleton className="h-2 w-16 mt-1" />
      </div>
    </div>
  </div>
);

const LastUsedColumnSkeleton = () => (
  <div className="px-1.5 rounded-md flex gap-2 items-center w-[140px] h-[22px] bg-grayA-3 animate-pulse">
    <Skeleton className="h-2 w-2 rounded-full" />
    <Skeleton className="h-2 w-12" />
    <Skeleton className="h-2 w-12" />
  </div>
);

const StatusColumnSkeleton = () => (
  <div
    className="flex w-25 items-center h-5.5 gap-2 px-1.5 py-1 rounded-md bg-grayA-3 animate-pulse"
    aria-busy="true"
    aria-live="polite"
  >
    <Skeleton className="h-2 w-2 rounded-full" />
    <Skeleton className="h-2 w-16" />
  </div>
);

const ActionColumnSkeleton = () => (
  <button
    type="button"
    className={cn(
      "group size-5 p-0 rounded-sm m-0 items-center flex justify-center animate-pulse",
      "border",
    )}
  >
    <IconDotsOutline12 className="text-gray-11" />
  </button>
);

type RenderApiKeySkeletonRowProps = {
  columns: DataTableColumnDef<KeyDetails>[];
  rowHeight: number;
};

export const renderApiKeySkeletonRow = ({ columns, rowHeight }: RenderApiKeySkeletonRowProps) =>
  columns.map((column, idx) => (
    <td
      key={column.id}
      className={cn(
        "text-xs align-middle whitespace-nowrap pr-4",
        idx === 0 ? "pl-0" : "",
        column.id === API_KEY_COLUMN_IDS.KEY ? "py-[6px]" : "py-1",
        column.meta?.cellClassName,
      )}
      style={{ height: `${rowHeight}px` }}
    >
      {column.id === API_KEY_COLUMN_IDS.KEY && <ApiKeyIdColumnSkeleton />}
      {column.id === API_KEY_COLUMN_IDS.VALUE && <KeyColumnSkeleton />}
      {column.id === API_KEY_COLUMN_IDS.USAGE && <UsageColumnSkeleton />}
      {column.id === API_KEY_COLUMN_IDS.LAST_USED && <LastUsedColumnSkeleton />}
      {column.id === API_KEY_COLUMN_IDS.STATUS && <StatusColumnSkeleton />}
      {column.id === API_KEY_COLUMN_IDS.ACTION && <ActionColumnSkeleton />}
    </td>
  ));
