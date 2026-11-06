"use client";

import { plural } from "@/lib/fmt";
import { IconCodeBranchOutline18, IconTableCodeOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { RelativeTime, Skeleton } from "@unkey/ui";
import { cn } from "cn";
import type { Database } from "./database-model";
import { DatabaseStatusBadge } from "./database-status-badge";
import { EngineTile, TILE_SIZE } from "./engine-icon";

export function DatabaseCard({
  db,
  onSelect,
}: { db: Database; onSelect: (databaseId: string) => void }) {
  const compute = db.planetscale.branches.production.compute;
  return (
    <div className="relative flex h-full min-h-[124px] w-full flex-col gap-4 rounded-lg border bg-raised p-5 shadow-xs transition-all duration-300 hover:border-strong">
      <button
        type="button"
        onClick={() => onSelect(db.id)}
        className="absolute inset-0 z-0 cursor-pointer rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
        aria-label={`View ${db.name}`}
      />
      <div className="flex min-h-5 items-center gap-2.5">
        <EngineTile engine={db.engine} size="md" />
        <span className="min-w-0 flex-1 truncate text-sm font-medium leading-5 text-gray-12">
          {db.name}
        </span>
        <span className="shrink-0 text-xs tabular-nums text-gray-10">
          {[compute?.size, db.region].filter(Boolean).join(" • ")}
        </span>
        <DatabaseStatusBadge status={db.status} />
      </div>
      {match(db)
        .with({ status: "ready" }, ({ instance }) => (
          <div className="flex flex-col gap-2 text-xs">
            <div className="flex items-center gap-4 text-gray-12">
              <span className="flex items-center gap-1.5">
                <IconTableCodeOutline18 className="size-3.5 text-gray-10" aria-hidden="true" />
                {plural(instance.tableCount, "table")}
              </span>
              <span className="flex items-center gap-1.5">
                <IconCodeBranchOutline18 className="size-3.5 text-gray-10" aria-hidden="true" />
                {plural(instance.branchCount, "branch", "branches")}
              </span>
            </div>
            <span className="text-gray-10">
              {instance.schemaUpdatedAt === null ? (
                "No schema yet"
              ) : (
                <>
                  Schema last updated <RelativeTime value={instance.schemaUpdatedAt} />
                </>
              )}
            </span>
          </div>
        ))
        .with({ status: "provisioning" }, () => (
          <span className="text-xs text-gray-10">Schema not ready yet</span>
        ))
        .exhaustive()}
    </div>
  );
}

export function DatabaseCardSkeleton() {
  return (
    <div className="flex h-full min-h-[124px] w-full flex-col gap-4 rounded-lg border bg-raised p-5 shadow-xs">
      <div className="flex min-h-5 items-center gap-2.5">
        <Skeleton className={cn("shrink-0", TILE_SIZE.md.tile)} />
        <Skeleton className="h-3.5 w-28" />
        <Skeleton className="ml-auto h-3 w-24 shrink-0" />
      </div>
      <div className="flex flex-col gap-2">
        <Skeleton className="h-3.5 w-36" />
        <Skeleton className="h-3 w-44" />
      </div>
    </div>
  );
}
