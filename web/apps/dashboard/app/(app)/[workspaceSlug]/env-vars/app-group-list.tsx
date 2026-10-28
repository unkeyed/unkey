"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { plural } from "@/lib/fmt";
import { IconTableCodeOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  ResourceListContent,
  Skeleton,
} from "@unkey/ui";
import { AppGroupCard } from "./app-group-card";
import { EMPTY_COPY, type EmptyReason, type VisibleAppGroup } from "./model";
import type { EnvVarOverviewView } from "./use-env-var-overview";

const SKELETON_CARDS = ["skeleton-a", "skeleton-b"];
const SKELETON_ROWS = Array.from({ length: 4 }, (_, index) => `row-${index}`);

export function AppGroupList({ view }: { view: EnvVarOverviewView }) {
  return match(view.list)
    .with({ status: "loading" }, () => <AppGroupListSkeleton />)
    .with({ status: "failed" }, ({ error, retry }) => (
      <LoadError title="Could not load environment variables" error={error} onRetry={retry} />
    ))
    .with({ status: "empty" }, ({ reason }) => (
      <AppGroupEmpty reason={reason} onClearFilters={view.clearFilters} />
    ))
    .with({ status: "ready" }, ({ rows, footer }) => {
      const resetKey = [view.search, ...Object.values(view.toolbar)].join("\n");
      return (
        <div className="flex flex-col gap-3">
          <output className="sr-only">{resultSummary(rows)}</output>
          {rows.map((group) => (
            <AppGroupCard key={`${group.app.id}\n${resetKey}`} group={group} search={view.search} />
          ))}
          <ListPager state={footer} noun="environment variables" />
        </div>
      );
    })
    .exhaustive();
}

function resultSummary(groups: readonly VisibleAppGroup[]): string {
  const variables = groups.reduce((total, group) => total + group.variables.length, 0);
  return `${plural(variables, "variable")} in ${plural(groups.length, "app")}`;
}

function AppGroupEmpty({
  reason,
  onClearFilters,
}: {
  reason: EmptyReason;
  onClearFilters: () => void;
}) {
  const copy = EMPTY_COPY[reason];

  return (
    <EmptyState>
      <EmptyStateIcon>
        <IconTableCodeOutline18 />
      </EmptyStateIcon>
      <EmptyStateHeader>
        <EmptyStateTitle>{copy.title}</EmptyStateTitle>
        <EmptyStateDescription>{copy.description}</EmptyStateDescription>
      </EmptyStateHeader>
      {copy.canClearFilters && (
        <EmptyStateActions>
          <Button variant="outline" onClick={onClearFilters}>
            Clear filters
          </Button>
        </EmptyStateActions>
      )}
    </EmptyState>
  );
}

function AppGroupListSkeleton() {
  return (
    <div className="flex flex-col gap-3" aria-busy="true">
      <output className="sr-only">Loading environment variables…</output>
      {SKELETON_CARDS.map((card) => (
        <ResourceListContent key={card} aria-hidden="true">
          <div className="flex h-11 items-center border-b px-4">
            <Skeleton className="h-3.5 w-40" />
          </div>
          <div className="py-1">
            {SKELETON_ROWS.map((row) => (
              <div
                key={row}
                className="grid h-8 grid-cols-[minmax(0,1.5fr)_minmax(0,1.5fr)_minmax(0,1fr)_64px] items-center gap-4 px-4"
              >
                <Skeleton className="h-3 w-48 max-w-full" />
                <Skeleton className="h-3 w-32 max-w-full" />
                <Skeleton className="h-3 w-20 max-w-full" />
                <Skeleton className="ml-auto h-3 w-12" />
              </div>
            ))}
          </div>
        </ResourceListContent>
      ))}
    </div>
  );
}
