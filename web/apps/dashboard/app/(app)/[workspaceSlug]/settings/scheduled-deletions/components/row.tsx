"use client";

import { type ScheduledDeletion, collection } from "@/lib/collections";
import { Button, TimestampInfo } from "@unkey/ui";

type RowProps = {
  row: ScheduledDeletion;
};

// Human-readable label per resource type. Keep in sync with the union
// in ScheduledDeletion; an unmatched type falls back to the raw value
// rather than throwing, so a backend that adds a new resource before
// the dashboard catches up still renders something sensible.
const resourceLabel: Record<ScheduledDeletion["resourceType"], string> = {
  project: "Project",
};

export function ScheduledDeletionRow({ row }: RowProps) {
  // Deleting from the collection triggers its onDelete, which calls the
  // restore mutation server-side and refetches. The row disappears from
  // the live query as the mutation succeeds; no local pending state to
  // manage here.
  const onRestore = () => {
    collection.scheduledDeletions.delete(`${row.resourceType}:${row.resourceId}`);
  };

  return (
    <div className="flex flex-col md:flex-row md:items-center px-4 py-3 gap-3 md:gap-0 transition-colors hover:bg-grayA-2">
      <div className="flex items-center justify-between md:contents">
        <div className="md:w-[35%] md:shrink-0 flex flex-col gap-1 min-w-0">
          <span className="text-[13px] text-accent-12 truncate font-semibold">{row.name}</span>
          <span className="text-xs text-gray-9">
            {resourceLabel[row.resourceType] ?? row.resourceType}
          </span>
        </div>

        <div className="md:w-[35%] md:shrink-0 flex items-start">
          <TimestampInfo
            value={row.deletePermanentlyAt}
            displayType="relative"
            side="top"
            align="start"
            className="text-[13px] text-gray-12"
          />
        </div>
      </div>

      <div className="md:w-[30%] md:shrink-0 flex items-center md:justify-end">
        <Button size="md" variant="outline" onClick={onRestore}>
          Restore
        </Button>
      </div>
    </div>
  );
}
