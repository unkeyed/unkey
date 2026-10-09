"use client";

import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { ResourceListBody, ResourceListContent, ResourceListItem } from "@unkey/ui";
import { useState } from "react";
import { useProjectData } from "../../../[appId]/(overview)/data-provider";
import { EnvVarRow, compareByName } from "./env-var-row";

export function SavedEnvVarsList({ envVars }: { envVars: EnvVar[] }) {
  const { environments } = useProjectData();
  const [editingId, setEditingId] = useState<string | null>(null);

  if (envVars.length === 0) {
    return null;
  }

  const environmentsById = new Map(environments.map((env) => [env.id, env]));
  const items = envVars
    .map((v) => ({ ...v, environment: environmentsById.get(v.environmentId) }))
    .sort(compareByName);

  return (
    <ResourceListContent>
      <ResourceListBody>
        {items.map((item) => (
          <ResourceListItem key={item.id}>
            <EnvVarRow
              item={item}
              searchQuery=""
              isEditing={editingId === item.id}
              onEdit={() => setEditingId(item.id)}
              onCloseEdit={() => setEditingId(null)}
            />
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}
