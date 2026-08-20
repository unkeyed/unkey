export type AppEnvironmentSelection = {
  appIds: Set<string>;
  environmentIds: Set<string>;
};

type EnvironmentIdsByApp = ReadonlyMap<string, readonly string[]>;
type AppEnvironmentField = "appId" | "environmentId";
type FilterLike = { field: string; value: string | number };

export function groupEnvironmentsByApp<T extends { appId: string; slug: string }>(
  environments: readonly T[],
): Map<string, T[]> {
  const grouped = new Map<string, T[]>();
  for (const environment of environments) {
    const appEnvironments = grouped.get(environment.appId) ?? [];
    appEnvironments.push(environment);
    grouped.set(environment.appId, appEnvironments);
  }
  for (const appEnvironments of grouped.values()) {
    appEnvironments.sort((left, right) => left.slug.localeCompare(right.slug));
  }
  return grouped;
}

export function getAppEnvironmentSelection(
  filters: readonly FilterLike[],
): AppEnvironmentSelection {
  return {
    appIds: new Set(
      filters.filter((filter) => filter.field === "appId").map((filter) => String(filter.value)),
    ),
    environmentIds: new Set(
      filters
        .filter((filter) => filter.field === "environmentId")
        .map((filter) => String(filter.value)),
    ),
  };
}

export function isEntireAppSelected(
  selection: AppEnvironmentSelection,
  appId: string,
  environmentIds: readonly string[],
): boolean {
  return (
    selection.appIds.has(appId) ||
    (environmentIds.length > 0 &&
      environmentIds.every((environmentId) => selection.environmentIds.has(environmentId)))
  );
}

export function toggleAppSelection(
  selection: AppEnvironmentSelection,
  appId: string,
  environmentIds: readonly string[],
): AppEnvironmentSelection {
  const appIds = new Set(selection.appIds);
  const selectedEnvironmentIds = new Set(selection.environmentIds);

  if (isEntireAppSelected(selection, appId, environmentIds)) {
    appIds.delete(appId);
  } else {
    appIds.add(appId);
  }
  for (const environmentId of environmentIds) {
    selectedEnvironmentIds.delete(environmentId);
  }

  return { appIds, environmentIds: selectedEnvironmentIds };
}

export function toggleEnvironmentSelection(
  selection: AppEnvironmentSelection,
  appId: string,
  environmentId: string,
  environmentIds: readonly string[],
): AppEnvironmentSelection {
  const appIds = new Set(selection.appIds);
  const selectedEnvironmentIds = new Set(selection.environmentIds);

  if (appIds.has(appId)) {
    for (const id of environmentIds) {
      selectedEnvironmentIds.add(id);
    }
    selectedEnvironmentIds.delete(environmentId);
  } else if (selectedEnvironmentIds.has(environmentId)) {
    selectedEnvironmentIds.delete(environmentId);
  } else {
    selectedEnvironmentIds.add(environmentId);
  }
  appIds.delete(appId);

  return { appIds, environmentIds: selectedEnvironmentIds };
}

export function createAppEnvironmentFilters<TFilter>(
  selection: AppEnvironmentSelection,
  environmentIdsByApp: EnvironmentIdsByApp,
  createFilter: (field: AppEnvironmentField, value: string) => TFilter,
): TFilter[] {
  const appIds = new Set(selection.appIds);
  const environmentIds = new Set(selection.environmentIds);

  for (const [appId, appEnvironmentIds] of environmentIdsByApp) {
    if (
      appEnvironmentIds.length > 0 &&
      appEnvironmentIds.every((environmentId) => environmentIds.has(environmentId))
    ) {
      appIds.add(appId);
      for (const environmentId of appEnvironmentIds) {
        environmentIds.delete(environmentId);
      }
    }
  }

  // The logs query intersects app and environment fields. Express mixed
  // selections as environment ids so whole apps and individual environments
  // form a union.
  if (environmentIds.size > 0) {
    for (const appId of appIds) {
      for (const environmentId of environmentIdsByApp.get(appId) ?? []) {
        environmentIds.add(environmentId);
      }
    }
    appIds.clear();
  }

  return [
    ...[...appIds].map((appId) => createFilter("appId", appId)),
    ...[...environmentIds].map((environmentId) => createFilter("environmentId", environmentId)),
  ];
}
