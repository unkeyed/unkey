"use client";

import { Planetscale } from "@unkey/icons";
import {
  Badge,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  Skeleton,
} from "@unkey/ui";
import type { ReactNode } from "react";
import { type Database, engineCopy, regionLabel, statusCopy } from "./database-model";

export function DatabasesTable({
  databases,
  onSelect,
}: {
  databases: readonly Database[];
  onSelect: (databaseId: string) => void;
}) {
  return (
    <DatabasesTableFrame>
      {databases.map((db) => (
        <DatabasesTableRow key={db.id} db={db} onSelect={onSelect} />
      ))}
    </DatabasesTableFrame>
  );
}

export function DatabasesTableSkeleton() {
  return (
    <DatabasesTableFrame busy>
      {["skeleton-1", "skeleton-2"].map((key) => (
        <ResourceListItem key={key} className="flex h-12 items-center gap-4 px-4">
          <Skeleton className="size-6 shrink-0 rounded-md" />
          <Skeleton className="h-3 w-28" />
          <Skeleton className="h-3 w-40" />
        </ResourceListItem>
      ))}
    </DatabasesTableFrame>
  );
}

function DatabasesTableFrame({ busy, children }: { busy?: boolean; children: ReactNode }) {
  return (
    <ResourceListContent
      aria-busy={busy}
      className="[--db-columns:minmax(0,1fr)_minmax(0,1.2fr)_minmax(0,1.2fr)_minmax(0,0.5fr)_minmax(0,0.6fr)]"
    >
      <div className="overflow-x-auto">
        <div className="min-w-[760px]">
          <div className="grid grid-cols-(--db-columns) items-center gap-4 border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12">
            <span>Database</span>
            <span>Engine</span>
            <span>Region</span>
            <span>Size</span>
            <span>Status</span>
          </div>
          <ResourceListBody aria-hidden={busy}>{children}</ResourceListBody>
        </div>
      </div>
    </ResourceListContent>
  );
}

function DatabasesTableRow({
  db,
  onSelect,
}: { db: Database; onSelect: (databaseId: string) => void }) {
  const status = statusCopy[db.status];
  return (
    <ResourceListItem className="grid grid-cols-(--db-columns) items-center gap-4 h-12 px-4 text-xs text-gray-12 transition-colors hover:bg-grayA-2">
      <button
        type="button"
        onClick={() => onSelect(db.id)}
        className="absolute inset-0 z-0 cursor-pointer"
        aria-label={`View ${db.name}`}
      />
      <span className="flex min-w-0 items-center gap-2.5">
        <span className="flex size-6 shrink-0 items-center justify-center rounded-md border bg-raised">
          <Planetscale className="size-3 text-gray-12" />
        </span>
        <span className="truncate text-sm font-medium">{db.name}</span>
      </span>
      <span className="truncate">{engineCopy[db.engine].label}</span>
      <span className="truncate text-gray-11">{regionLabel[db.region]}</span>
      <span className="truncate font-mono text-gray-11">
        {db.planetscale.branches.production.compute?.size}
      </span>
      <span>
        <Badge variant={status.badge} size="sm">
          {status.label}
        </Badge>
      </span>
    </ResourceListItem>
  );
}
