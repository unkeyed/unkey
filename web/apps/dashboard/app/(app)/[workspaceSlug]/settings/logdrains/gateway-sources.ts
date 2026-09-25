export type SourceApp = { id: string; name: string; environments: { id: string; name: string }[] };
export type SourceProject = { id: string; name: string; apps: SourceApp[] };

export type SourceFilters = {
  projectIds: string[];
  appIds: string[];
  environmentIds: string[];
};

export function buildSourceTree(
  projects: { id: string; name: string; apps: { id: string; name: string }[] }[],
  environments: { id: string; name: string; appId: string }[],
): SourceProject[] {
  return projects.map((project) => ({
    id: project.id,
    name: project.name,
    apps: project.apps.map((app) => ({
      id: app.id,
      name: app.name,
      environments: environments
        .filter((environment) => environment.appId === app.id)
        .map((environment) => ({ id: environment.id, name: environment.name })),
    })),
  }));
}

export function environmentIdsOf(tree: SourceProject[]): string[] {
  return tree.flatMap((project) =>
    project.apps.flatMap((app) => app.environments.map((environment) => environment.id)),
  );
}

/**
 * The drain filters are ANDed server side, so a stored config selects exactly the
 * environments that survive all three lists. An empty config means every
 * environment while the drain is unfiltered, and none once the form says the
 * user is picking sources.
 */
export function tickedEnvironmentIds(
  tree: SourceProject[],
  mode: "all" | "some",
  filters: SourceFilters,
): string[] {
  if (mode === "all") {
    return environmentIdsOf(tree);
  }
  if (filters.projectIds.length + filters.appIds.length + filters.environmentIds.length === 0) {
    return [];
  }
  return selectedEnvironmentIds(tree, filters);
}

export function selectedEnvironmentIds(tree: SourceProject[], filters: SourceFilters): string[] {
  return tree.flatMap((project) =>
    project.apps.flatMap((app) =>
      app.environments
        .filter(
          (environment) =>
            (filters.projectIds.length === 0 || filters.projectIds.includes(project.id)) &&
            (filters.appIds.length === 0 || filters.appIds.includes(app.id)) &&
            (filters.environmentIds.length === 0 ||
              filters.environmentIds.includes(environment.id)),
        )
        .map((environment) => environment.id),
    ),
  );
}

/**
 * Whole projects and whole apps stay addressed by their own id, so environments
 * added later keep flowing into the drain. Anything else has to be pinned to
 * environment ids, because the three lists are ANDed and cannot express a union.
 */
export function encodeSources(tree: SourceProject[], selected: Set<string>): SourceFilters {
  const empty = { projectIds: [], appIds: [], environmentIds: [] };
  if (selected.size === 0) {
    return empty;
  }

  const wholeProjects = tree.filter((project) => {
    const ids = project.apps.flatMap((app) =>
      app.environments.map((environment) => environment.id),
    );
    return ids.length > 0 && ids.every((id) => selected.has(id));
  });
  const projectReach = wholeProjects.flatMap((project) =>
    project.apps.flatMap((app) => app.environments.map((environment) => environment.id)),
  );
  if (projectReach.length === selected.size) {
    return { ...empty, projectIds: wholeProjects.map((project) => project.id) };
  }

  const wholeApps = tree
    .flatMap((project) => project.apps)
    .filter(
      (app) =>
        app.environments.length > 0 &&
        app.environments.every((environment) => selected.has(environment.id)),
    );
  const appReach = wholeApps.flatMap((app) =>
    app.environments.map((environment) => environment.id),
  );
  if (appReach.length === selected.size) {
    return { ...empty, appIds: wholeApps.map((app) => app.id) };
  }

  return {
    ...empty,
    environmentIds: environmentIdsOf(tree).filter((id) => selected.has(id)),
  };
}
