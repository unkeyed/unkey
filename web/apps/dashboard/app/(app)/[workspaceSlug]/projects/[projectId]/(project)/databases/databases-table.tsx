"use client";

import { ListTableRow, listTable } from "@/components/list-views";
import { Badge } from "@unkey/ui";
import { type Database, engineCopy, regionLabel, statusCopy } from "./database-model";
import { EngineTile } from "./engine-icon";

export const DATABASE_TABLE = listTable({
  minWidth: 760,
  columns: [
    { id: "database", header: "Database", width: "minmax(0,1fr)", skeleton: "title" },
    { id: "engine", header: "Engine", width: "minmax(0,1.2fr)", skeleton: "text" },
    { id: "region", header: "Region", width: "minmax(0,1.2fr)", skeleton: "text" },
    { id: "size", header: "Size", width: "minmax(0,0.5fr)", skeleton: "text" },
    { id: "status", header: "Status", width: "minmax(0,0.6fr)", skeleton: "text" },
  ],
});

export function DatabaseRow({
  db,
  onSelect,
}: { db: Database; onSelect: (databaseId: string) => void }) {
  const status = statusCopy[db.status];
  return (
    <ListTableRow
      table={DATABASE_TABLE}
      label={`View ${db.name}`}
      link={<button type="button" onClick={() => onSelect(db.id)} className="cursor-pointer" />}
      cells={{
        database: (
          <span className="flex min-w-0 items-center gap-2.5">
            <EngineTile engine={db.engine} size="sm" />
            <span className="truncate text-sm font-medium">{db.name}</span>
          </span>
        ),
        engine: <span className="truncate">{engineCopy[db.engine].label}</span>,
        region: <span className="truncate text-gray-11">{regionLabel[db.region]}</span>,
        size: (
          <span className="truncate font-mono text-gray-11">
            {db.planetscale.branches.production.compute?.size}
          </span>
        ),
        status: (
          <span>
            <Badge variant={status.tableBadge} size="sm">
              {status.label}
            </Badge>
          </span>
        ),
      }}
    />
  );
}
