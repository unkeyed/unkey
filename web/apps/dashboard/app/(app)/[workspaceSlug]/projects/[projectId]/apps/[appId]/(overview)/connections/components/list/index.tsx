"use client";

import { ResourceListBody } from "@unkey/ui";
import type { ConnectionTargets } from "../../connection-rules";
import type { ConnectionEnvs, MergedConnection } from "../../model/merge";
import { ConnectionRow, ConnectionsListHeader } from "./row";

export function ConnectionsList({
  envs,
  merged,
  targets,
  onEdit,
  onRemove,
}: {
  envs: ConnectionEnvs;
  merged: MergedConnection[];
  targets: ConnectionTargets;
  onEdit: (targetAppId: string) => void;
  onRemove: (connection: MergedConnection) => void;
}) {
  return (
    <div>
      <ConnectionsListHeader />
      <ResourceListBody>
        {merged.map((connection) => (
          <ConnectionRow
            key={connection.targetAppId}
            connection={connection}
            envs={envs}
            targets={targets}
            onEdit={onEdit}
            onRemove={onRemove}
          />
        ))}
      </ResourceListBody>
    </div>
  );
}
