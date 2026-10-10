"use client";

import { LoadError } from "@/components/load-error";
import { useResourceSearch } from "@/components/resource-search-input";
import type { Environment } from "@/lib/collections/deploy/environments";
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
import { ALL_ENVIRONMENTS } from "../../environment-select";
import { ENV_VARS_SEARCH_KEY, type SortOption } from "../toolbar/env-vars-toolbar";
import { useAppEnvVars } from "../use-app-env-vars";
import { type EnvVarItem, EnvVarRow, compareByName, withEnvironments } from "./env-var-row";
import { EnvVarSelectionBar } from "./env-var-selection-bar";
import { EnvVarsSkeleton } from "./env-vars-skeleton";
import { useRowSelection } from "./use-row-selection";
import { useVirtualList } from "./use-virtual-list";

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

  const { data: envVarData, isLoading, failed, retry } = useAppEnvVars(projectId, appId);

  const rows = useMemo((): EnvVarItem[] => {
    if (!envVarData) {
      return [];
    }
    const query = searchQuery.toLowerCase();
    const filtered = envVarData.filter(
      (v) =>
        (!query || v.key.toLowerCase().includes(query)) &&
        (environmentFilter === ALL_ENVIRONMENTS || v.environmentId === environmentFilter),
    );

    return withEnvironments(filtered, environments).sort(
      sortBy === "name-asc"
        ? compareByName
        : (a, b) => b.createdAt - a.createdAt || compareByName(a, b),
    );
  }, [envVarData, environments, searchQuery, environmentFilter, sortBy]);

  if (failed) {
    return <LoadError title="Could not load environment variables" onRetry={retry} />;
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
    <EnvVarRows
      key={`${searchQuery}|${environmentFilter}`}
      rows={rows}
      searchQuery={searchQuery}
      editingId={editingId}
      onEdit={setEditingId}
      onCloseEdit={closeEdit}
    />
  );
}

function EnvVarRows({
  rows,
  searchQuery,
  editingId,
  onEdit,
  onCloseEdit,
}: {
  rows: EnvVarItem[];
  searchQuery: string;
  editingId: string | null;
  onEdit: (id: string) => void;
  onCloseEdit: () => void;
}) {
  const selection = useRowSelection(rows);
  const { virtualizer, listRefCallback, scrollMargin } = useVirtualList(rows);

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
                  onEdit={() => onEdit(item.id)}
                  onCloseEdit={onCloseEdit}
                  selection={{
                    isSelected: selection.selectedIds.has(item.id),
                    hasSelection: selection.selected.length > 0,
                    onToggle: (shiftKey) => selection.toggleRowSelection(item.id, shiftKey),
                  }}
                />
              </ResourceListItem>
            );
          })}
        </ResourceListBody>
      </ResourceListContent>
      <EnvVarSelectionBar
        selected={selection.selected}
        onClearSelection={selection.clearSelection}
      />
    </>
  );
}
