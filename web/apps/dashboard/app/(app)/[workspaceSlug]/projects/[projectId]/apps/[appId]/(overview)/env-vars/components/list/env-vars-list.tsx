"use client";

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
import { useCallback, useDeferredValue, useMemo, useState } from "react";
import { LoadError } from "../../../../components/load-error";
import { useRowSelection } from "../../hooks/use-row-selection";
import { useVirtualList } from "../../hooks/use-virtual-list";
import { EnvVarsSkeleton } from "../shared/env-vars-skeleton";
import type { EnvironmentFilter, SortOption } from "../toolbar/env-vars-toolbar";
import { type EnvVarItem, EnvVarRow } from "./env-var-row";
import { EnvVarSelectionBar } from "./env-var-selection-bar";

type EnvVarsListProps = {
  projectId: string;
  appId: string;
  environments: Environment[];
  searchQuery: string;
  environmentFilter: EnvironmentFilter;
  sortBy: SortOption;
};

export function EnvVarsList({
  projectId,
  appId,
  environments,
  searchQuery,
  environmentFilter,
  sortBy,
}: EnvVarsListProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const closeEdit = useCallback(() => setEditingId(null), []);

  const openEdit = useCallback(
    (id: string) => {
      if (editingId !== null && editingId !== id) {
        setEditingId(null);
        requestAnimationFrame(() => setEditingId(id));
      } else {
        setEditingId(id);
      }
    },
    [editingId],
  );
  const deferredQuery = useDeferredValue(searchQuery);

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
    const query = deferredQuery.toLowerCase();
    const environmentsById = new Map(environments.map((env) => [env.id, env]));

    const filtered = envVarData
      .filter(
        (v) =>
          (!query || v.key.toLowerCase().includes(query)) &&
          (environmentFilter === "all" || v.environmentId === environmentFilter),
      )
      .map((v) => ({ ...v, environment: environmentsById.get(v.environmentId) }));

    const byName = (a: EnvVarItem, b: EnvVarItem) =>
      a.key.localeCompare(b.key) ||
      (a.environment?.slug ?? "").localeCompare(b.environment?.slug ?? "");
    return filtered.sort(
      sortBy === "name-asc" ? byName : (a, b) => b.createdAt - a.createdAt || byName(a, b),
    );
  }, [envVarData, environments, deferredQuery, environmentFilter, sortBy]);

  const {
    selectedIds,
    toggleRowSelection,
    handleBulkDelete,
    handleBulkMakeSensitive,
    clearSelection,
  } = useRowSelection(rows, envVarData);

  const { virtualizer, listRefCallback, scrollMargin } = useVirtualList(rows, editingId);
  const envVarsLoad = useCollectionLoad(collection.envVars.utils);

  if (isLoading) {
    return <EnvVarsSkeleton />;
  }

  if (envVarsLoad.failed) {
    return <LoadError title="Could not load environment variables" onRetry={envVarsLoad.retry} />;
  }

  if (rows.length === 0) {
    const isFiltered = searchQuery !== "" || environmentFilter !== "all";
    return (
      <EmptyState>
        <EmptyStateIcon>
          <IconTableCodeOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader>
          <EmptyStateTitle>
            {isFiltered ? "No matching variables" : "No environment variables"}
          </EmptyStateTitle>
          <EmptyStateDescription>
            {searchQuery
              ? `No variables matching "${searchQuery}". Try a different search term.`
              : isFiltered
                ? "No variables in this environment."
                : "Add a variable to pass secrets and settings to your deployments."}
          </EmptyStateDescription>
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
                  searchQuery={deferredQuery}
                  isEditing={editingId === item.id}
                  onEdit={() => openEdit(item.id)}
                  onCloseEdit={closeEdit}
                  isSelected={selectedIds.has(item.id)}
                  hasSelection={selectedIds.size > 0}
                  onToggleSelection={(shiftKey) => toggleRowSelection(virtualRow.index, shiftKey)}
                />
              </ResourceListItem>
            );
          })}
        </ResourceListBody>
      </ResourceListContent>
      <EnvVarSelectionBar
        selectedCount={selectedIds.size}
        onDelete={handleBulkDelete}
        onMakeSensitive={handleBulkMakeSensitive}
        onClearSelection={clearSelection}
      />
    </>
  );
}
