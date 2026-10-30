"use client";

import { ResourceListBody } from "@unkey/ui";
import type { ReactNode } from "react";
import type { ConnectionRowModel } from "../../hooks/use-connections-data";
import type { ListedConnection } from "../../model/target";
import { ConnectionRow, ConnectionsListHeader } from "./row";

export function ConnectionsList({
  rows,
  onEdit,
  onRemove,
  removing,
  removed,
}: {
  rows: readonly ConnectionRowModel[];
  onEdit: (connection: ListedConnection) => void;
  onRemove: (connection: ListedConnection) => void;
  removing: boolean;
  removed: ReactNode;
}) {
  return (
    <div>
      <ConnectionsListHeader />
      <ResourceListBody>
        {rows.map((row) => (
          <ConnectionRow
            key={row.connection.id}
            row={row}
            onEdit={onEdit}
            onRemove={onRemove}
            removing={removing}
          />
        ))}
        {removed}
      </ResourceListBody>
    </div>
  );
}
