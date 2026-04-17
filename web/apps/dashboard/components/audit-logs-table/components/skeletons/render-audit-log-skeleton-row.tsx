import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import { CreatedAtColumnSkeleton } from "@unkey/ui";
import { AUDIT_LOG_COLUMN_IDS } from "../../columns/create-audit-log-columns";

type RenderAuditLogSkeletonRowProps = {
  columns: DataTableColumnDef<AuditLog>[];
  rowHeight: number;
};

const TextSkeleton = ({ width }: { width: string }) => (
  <div className={cn("h-3 rounded bg-grayA-3 animate-pulse")} style={{ width }} />
);

const BadgeSkeleton = () => <div className="h-5 w-14 rounded-md bg-grayA-3 animate-pulse" />;

export const renderAuditLogSkeletonRow = ({ columns }: RenderAuditLogSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
    >
      {column.id === AUDIT_LOG_COLUMN_IDS.TIME.id && (
        <div className="pl-2">
          <CreatedAtColumnSkeleton />
        </div>
      )}
      {column.id === AUDIT_LOG_COLUMN_IDS.ACTOR.id && <TextSkeleton width="70%" />}
      {column.id === AUDIT_LOG_COLUMN_IDS.ACTION.id && <BadgeSkeleton />}
      {column.id === AUDIT_LOG_COLUMN_IDS.EVENT.id && <TextSkeleton width="80%" />}
      {column.id === AUDIT_LOG_COLUMN_IDS.DESCRIPTION.id && <TextSkeleton width="90%" />}
    </td>
  ));
