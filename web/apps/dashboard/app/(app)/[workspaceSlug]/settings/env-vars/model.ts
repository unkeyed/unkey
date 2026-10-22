import { environmentName } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { type EnvVarType, TYPE_MARKERS } from "@/components/deploy/env-vars/type-marker";
import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { IconBarsFilterOutline18, IconLayers3Outline18, IconLockOutline18 } from "@unkey/icons";
import { formatTimestamp } from "@unkey/ui";
import type { ComponentType } from "react";

type ListedAppGroup = inferRouterOutputs<Router>["deploy"]["envVar"]["list"][number];
type ListedVariable = ListedAppGroup["variables"][number];

export type VisibleVariable = ListedVariable & { changedAt: number };
export type VisibleAppGroup = Omit<ListedAppGroup, "variables"> & { variables: VisibleVariable[] };

type TypeFilter = "all" | EnvVarType;
type EnvironmentFilter = "all" | EnvironmentKind;
type SortOption = "updated" | "name" | "type";

type Option<T extends string> = { value: T; label: string };

const TYPE_FILTER_OPTIONS = [
  { value: "all", label: "All types" },
  { value: "writeonly", label: TYPE_MARKERS.writeonly.name },
  { value: "recoverable", label: TYPE_MARKERS.recoverable.name },
] as const satisfies readonly Option<TypeFilter>[];

const ENVIRONMENT_FILTER_OPTIONS = [
  { value: "all", label: "All environments" },
  { value: "production", label: environmentName("production") },
  { value: "preview", label: environmentName("preview") },
] as const satisfies readonly Option<EnvironmentFilter>[];

const SORT_OPTIONS = [
  { value: "updated", label: "Last updated" },
  { value: "name", label: "Name" },
  { value: "type", label: "Type" },
] as const satisfies readonly Option<SortOption>[];

export type ToolbarState = {
  type: TypeFilter;
  environment: EnvironmentFilter;
  sort: SortOption;
};

type ToolbarField<T extends string> = {
  label: string;
  icon: ComponentType<{ className?: string }>;
  options: readonly [Option<T>, ...Option<T>[]];
};

export const TOOLBAR_FIELDS: { [K in keyof ToolbarState]: ToolbarField<ToolbarState[K]> } = {
  type: { label: "Filter by type", icon: IconLockOutline18, options: TYPE_FILTER_OPTIONS },
  environment: {
    label: "Filter by environment",
    icon: IconLayers3Outline18,
    options: ENVIRONMENT_FILTER_OPTIONS,
  },
  sort: { label: "Sort", icon: IconBarsFilterOutline18, options: SORT_OPTIONS },
};

type OverviewFilter = ToolbarState & { search: string };

const TYPE_ORDER: Record<EnvVarType, number> = { writeonly: 0, recoverable: 1 };

const byKey = (a: VisibleVariable, b: VisibleVariable) => a.key.localeCompare(b.key);

const COMPARATORS: Record<SortOption, (a: VisibleVariable, b: VisibleVariable) => number> = {
  updated: (a, b) => b.changedAt - a.changedAt || byKey(a, b),
  name: (a, b) => byKey(a, b) || TYPE_ORDER[a.type] - TYPE_ORDER[b.type],
  type: (a, b) => TYPE_ORDER[a.type] - TYPE_ORDER[b.type] || byKey(a, b),
};

export type EmptyReason = "none" | "no-match";

export const EMPTY_COPY: Record<
  EmptyReason,
  { title: string; description: string; canClearFilters: boolean }
> = {
  none: {
    title: "No environment variables",
    description: "Variables you add to an app appear here, grouped by app.",
    canClearFilters: false,
  },
  "no-match": {
    title: "No matching variables",
    description: "No variable matches the search and filters.",
    canClearFilters: true,
  },
};

export const PAGE_SIZE = 15;

export type PageWindow = {
  current: number;
  start: number;
  end: number;
  hasPrevious: boolean;
  hasNext: boolean;
  rangeLabel: string;
};

export function pageWindow(total: number, page: number): PageWindow {
  const lastPage = Math.max(0, Math.ceil(total / PAGE_SIZE) - 1);
  const current = Math.min(Math.max(page, 0), lastPage);
  const start = current * PAGE_SIZE;
  const end = Math.min(start + PAGE_SIZE, total);
  return {
    current,
    start,
    end,
    hasPrevious: current > 0,
    hasNext: current < lastPage,
    rangeLabel: `${start + 1}–${end} of ${total}`,
  };
}

export function selectAppGroups(
  groups: ListedAppGroup[],
  filter: OverviewFilter,
): VisibleAppGroup[] {
  const search = filter.search.toLowerCase();
  const selected: VisibleAppGroup[] = [];

  for (const group of groups) {
    const variables = group.variables
      .filter(
        (variable) =>
          (filter.type === "all" || variable.type === filter.type) &&
          variable.key.toLowerCase().includes(search),
      )
      .map((variable) => {
        const environments = variable.environments.filter(
          (environment) => filter.environment === "all" || environment.kind === filter.environment,
        );
        return {
          ...variable,
          environments,
          changedAt: Math.max(...environments.map((environment) => environment.changedAt)),
        };
      })
      .filter((variable) => variable.environments.length > 0)
      .sort(COMPARATORS[filter.sort]);
    if (variables.length > 0) {
      selected.push({ ...group, variables });
    }
  }

  return selected;
}

const environmentList = new Intl.ListFormat("en", { type: "conjunction" });

export function environmentProse(environments: VisibleVariable["environments"]): string {
  return environmentList.format(
    environments.map((environment) => environmentName(environment.slug)),
  );
}

export function formatChangedAt(millis: number, now: number): string {
  const sameYear = new Date(millis).getFullYear() === new Date(now).getFullYear();
  return formatTimestamp(millis, sameYear ? "MMM d" : "M/d/yy");
}
