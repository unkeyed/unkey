"use client";

import { useResourceSearch } from "@/components/resource-search-input";
import { type ListView, listOf } from "@/lib/loadable/list";
import { type Loadable, fromQuery } from "@/lib/loadable/loadable";
import { trpc } from "@/lib/trpc/client";
import { parseAsStringLiteral, useQueryStates } from "nuqs";
import { useMemo } from "react";
import {
  type EmptyReason,
  TOOLBAR_FIELDS,
  type ToolbarState,
  type VisibleAppGroup,
  selectAppGroups,
} from "./model";

export const OVERVIEW_SEARCH_KEY = "search";

function fieldParser<V extends string>(options: readonly [{ value: V }, ...{ value: V }[]]) {
  return parseAsStringLiteral(options.map((option) => option.value)).withDefault(options[0].value);
}

const toolbarParsers = {
  type: fieldParser(TOOLBAR_FIELDS.type.options),
  environment: fieldParser(TOOLBAR_FIELDS.environment.options),
  sort: fieldParser(TOOLBAR_FIELDS.sort.options),
};

export type AppGroupListView =
  | Exclude<ListView<VisibleAppGroup>, { status: "empty" }>
  | { status: "empty"; reason: EmptyReason };

export type EnvVarOverviewView = {
  list: AppGroupListView;
  search: string;
  toolbar: ToolbarState;
  setToolbar: (patch: Partial<ToolbarState>) => void;
  clearFilters: () => void;
};

export function useEnvVarOverview(): EnvVarOverviewView {
  const query = trpc.deploy.envVar.list.useQuery(undefined, { refetchOnMount: "always" });
  const envVars = fromQuery(query);
  const [search, setSearch] = useResourceSearch(OVERVIEW_SEARCH_KEY);
  const [toolbar, setToolbar] = useQueryStates(toolbarParsers, {
    history: "replace",
    shallow: true,
    clearOnDefault: true,
  });

  const listed = envVars.status === "ready" ? envVars.data : undefined;
  const groups = useMemo(
    () => selectAppGroups(listed ?? [], { search, ...toolbar }),
    [listed, search, toolbar],
  );

  return {
    list: appGroupListOf(envVars, groups),
    search,
    toolbar,
    setToolbar,
    clearFilters: () => {
      setSearch(null);
      setToolbar({ type: null, environment: null });
    },
  };
}

function appGroupListOf<T>(
  envVars: Loadable<readonly T[]>,
  groups: readonly VisibleAppGroup[],
): AppGroupListView {
  if (envVars.status !== "ready") {
    return envVars;
  }
  const view = listOf({ ...envVars, data: groups });
  if (view.status !== "empty") {
    return view;
  }
  return { status: "empty", reason: envVars.data.length > 0 ? "no-match" : "none" };
}
