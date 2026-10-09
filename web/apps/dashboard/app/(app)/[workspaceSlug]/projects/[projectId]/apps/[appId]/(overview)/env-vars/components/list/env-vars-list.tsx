"use client";

import { LoadError } from "@/components/load-error";
import { useResourceSearch } from "@/components/resource-search-input";
import { collection } from "@/lib/collections";
import type { Environment } from "@/lib/collections/deploy/environments";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { IconTableCodeOutline18 } from "@unkey/icons";
import {
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
} from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";
import {
  type EnvVarItem,
  EnvVarRow,
  compareByName,
} from "../../../../../_components/env-vars/list/env-var-row";
import { ALL_ENVIRONMENTS } from "../../../../../_components/env-vars/shared/environment-select";
import { useRowSelection } from "../../hooks/use-row-selection";
import { useVirtualList } from "../../hooks/use-virtual-list";
import { EnvVarsSkeleton } from "../shared/env-vars-skeleton";
import { ENV_VARS_SEARCH_KEY, type SortOption } from "../toolbar/env-vars-toolbar";
import { EnvVarSelectionBar } from "./env-var-selection-bar";

type EmptyReason = "search" | "environment" | "none";

const EMPTY_COPY: Record<EmptyReason, { title: string; description: (query: string) => string }> = {
  search: {
    title: "No matching variables",
    description: (query) => `No variables matching "${query}". Try a different search term.`,
  },
  environment: {
    title: "No matching variables",
    description: () => "No variables in this environment.",
  },
  none: {
    title: "No environment variables",
    description: () => "Add a variable to pass secrets and settings to your deployments.",
  },
};

function emptyReason(query: string, environmentFilter: string): EmptyReason {
  if (query) {
    return "search";
  }
  return environmentFilter === ALL_ENVIRONMENTS ? "none" : "environment";
}

type EnvVarsListProps = {
  projectId: string;
  appId: string;
  environments: Environment[];
  environmentFilter: string;
  sortBy: SortOption;
};

export function EnvVarsList({
  projectId,
  appId,
  environments,
  environmentFilter,
  sortBy,
}: EnvVarsListProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const closeEdit = useCallback(() => setEditingId(null), []);
  const [searchQuery] = useResourceSearch(ENV_VARS_SEARCH_KEY);

  const { data: envVarData, isLoading } = useLiveQuery(
    (q) =>
      q
        .from({ v: collection.envVars })
        .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId))),
    [projectId, appId],
  );

  const rows = useMemo((): EnvVarItem[] => {
    if (!envVarData) {
      return [];
    }
    const query = searchQuery.toLowerCase();
    const environmentsById = new Map(environments.map((env) => [env.id, env]));

    const filtered = envVarData
      .filter(
        (v) =>
          (!query || v.key.toLowerCase().includes(query)) &&
          (environmentFilter === ALL_ENVIRONMENTS || v.environmentId === environmentFilter),
      )
      .map((v) => ({ ...v, environment: environmentsById.get(v.environmentId) }));

    return filtered.sort(
      sortBy === "name-asc"
        ? compareByName
        : (a, b) => b.createdAt - a.createdAt || compareByName(a, b),
    );
  }, [envVarData, environments, searchQuery, environmentFilter, sortBy]);

  const selection = useRowSelection(rows, JSON.stringify([searchQuery, environmentFilter]));

  const { virtualizer, listRefCallback, scrollMargin } = useVirtualList(rows);
  const envVarsLoad = useCollectionLoad(collection.envVars.utils);

  if (envVarsLoad.failed) {
    return <LoadError title="Could not load environment variables" onRetry={envVarsLoad.retry} />;
  }

  if (isLoading) {
    return <EnvVarsSkeleton />;
  }

  if (rows.length === 0) {
    const copy = EMPTY_COPY[emptyReason(searchQuery, environmentFilter)];
    return (
      <EmptyState>
        <EmptyStateIcon>
          <IconTableCodeOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader>
          <EmptyStateTitle>{copy.title}</EmptyStateTitle>
          <EmptyStateDescription>{copy.description(searchQuery)}</EmptyStateDescription>
        </EmptyStateHeader>
      </EmptyState>
    );
  }

  return (
    <>
      <ResourceListContent ref={listRefCallback}>
        <ResourceListBody style={{ height: virtualizer.getTotalSize(), position: "relative" }}>
          {virtualizer.getVirtualItems().map((virtualRow) => {
            const item = rows[virtualRow.index];
            return (
              <ResourceListItem
                key={virtualRow.key}
                ref={virtualizer.measureElement}
                data-index={virtualRow.index}
                style={{
                  position: "absolute",
                  top: 0,
                  left: 0,
                  width: "100%",
                  transform: `translateY(${virtualRow.start - scrollMargin}px)`,
                }}
              >
                <EnvVarRow
                  item={item}
                  searchQuery={searchQuery}
                  isEditing={editingId === item.id}
                  onEdit={() => setEditingId(item.id)}
                  onCloseEdit={closeEdit}
                  selection={{
                    isSelected: selection.selectedIds.has(item.id),
                    hasSelection: selection.selectedCount > 0,
                    onToggle: (shiftKey) => selection.toggleRowSelection(item.id, shiftKey),
                  }}
                />
              </ResourceListItem>
            );
          })}
        </ResourceListBody>
      </ResourceListContent>
      <EnvVarSelectionBar
        selectedCount={selection.selectedCount}
        isMakingSensitive={selection.isMakingSensitive}
        onDelete={selection.handleBulkDelete}
        onMakeSensitive={selection.handleBulkMakeSensitive}
        onClearSelection={selection.clearSelection}
      />
    </>
  );
}
