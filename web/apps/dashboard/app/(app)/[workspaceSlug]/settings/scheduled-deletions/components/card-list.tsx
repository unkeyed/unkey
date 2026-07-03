"use client";

import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { Empty } from "@unkey/ui";
import { ScheduledDeletionRow } from "./row";
import { ScheduledDeletionsSkeleton } from "./skeleton";

export function ScheduledDeletionsCardList() {
  const rows = useLiveQuery((q) =>
    q.from({ row: collection.scheduledDeletions }).orderBy(({ row }) => row.deletePermanentlyAt),
  );

  if (rows.isLoading) {
    return <ScheduledDeletionsSkeleton />;
  }

  if (rows.data.length === 0) {
    return (
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
        <div className="w-full flex justify-center items-center py-16 px-4">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Nothing scheduled</Empty.Title>
            <Empty.Description className="text-left">
              Resources you delete will appear here during their grace period and can be restored
              before they are permanently removed.
            </Empty.Description>
          </Empty>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
      {rows.data.map((row) => (
        <ScheduledDeletionRow key={`${row.resourceType}:${row.resourceId}`} row={row} />
      ))}
    </div>
  );
}
