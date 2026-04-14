import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  ActionColumnSkeleton,
  CreatedAtColumnSkeleton,
  LastUpdatedColumnSkeleton,
} from "@unkey/ui";
import { Fingerprint } from "@unkey/icons";
import type { z } from "zod";
import { IDENTITY_COLUMN_IDS } from "../../columns/create-identities-columns";

type Identity = z.infer<typeof IdentityResponseSchema>;

const IdentityExternalIdSkeleton = () => (
  <div className="flex flex-col items-start px-[18px] py-[6px]">
    <div className="flex gap-4 items-center">
      <div className="size-5 rounded-sm flex items-center justify-center border border-grayA-3 bg-grayA-3 animate-pulse">
        <Fingerprint iconSize="md-medium" className="text-gray-12 opacity-50" />
      </div>
      <div className="flex flex-col gap-1">
        <div className="h-2 w-40 bg-grayA-3 rounded-sm animate-pulse" />
        <div className="h-2 w-16 bg-grayA-3 rounded-sm animate-pulse mt-1" />
      </div>
    </div>
  </div>
);

const CountSkeleton = () => (
  <div className="flex items-center px-3 py-1">
    <div className="h-2 w-8 bg-grayA-3 rounded-sm animate-pulse" />
  </div>
);

type RenderIdentitiesSkeletonRowProps = {
  columns: DataTableColumnDef<Identity>[];
  rowHeight: number;
};

export const renderIdentitiesSkeletonRow = ({
  columns,
}: RenderIdentitiesSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
    >
      {column.id === IDENTITY_COLUMN_IDS.EXTERNAL_ID && <IdentityExternalIdSkeleton />}
      {column.id === IDENTITY_COLUMN_IDS.KEYS && <CountSkeleton />}
      {column.id === IDENTITY_COLUMN_IDS.RATELIMITS && <CountSkeleton />}
      {column.id === IDENTITY_COLUMN_IDS.CREATED && (
        <div className="px-3 py-1">
          <CreatedAtColumnSkeleton />
        </div>
      )}
      {column.id === IDENTITY_COLUMN_IDS.LAST_USED && (
        <div className="px-3 py-1">
          <LastUpdatedColumnSkeleton />
        </div>
      )}
      {column.id === IDENTITY_COLUMN_IDS.ACTION && (
        <div className="flex items-center justify-end px-3 py-1">
          <ActionColumnSkeleton />
        </div>
      )}
    </td>
  ));
