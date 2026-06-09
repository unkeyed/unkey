/**
 * Single source of truth for dashboard route paths in the /projects area.
 * Every builder takes the minimal scope object it needs and returns a string.
 */
type WorkspaceScope = { workspaceSlug: string };
type ProjectScope = WorkspaceScope & { projectId: string };
type AppScope = ProjectScope & { appId: string };

export function projectsPath({
  workspaceSlug,
  new: isNew,
}: WorkspaceScope & { new?: boolean }): string {
  const base = `/${workspaceSlug}/projects`;
  return isNew ? `${base}?new=true` : base;
}

export function projectPath({ workspaceSlug, projectId }: ProjectScope): string {
  return `${projectsPath({ workspaceSlug })}/${projectId}`;
}

export function projectSettingsPath(scope: ProjectScope): string {
  return `${projectPath(scope)}/settings`;
}

export function projectLogsPath({ appId, ...scope }: ProjectScope & { appId?: string }): string {
  const base = `${projectPath(scope)}/logs`;
  return appId ? `${base}?appId=${appId}` : base;
}

export function projectRequestsPath({
  since,
  appId,
  deploymentId,
  ...scope
}: ProjectScope & { since?: string; appId?: string; deploymentId?: string }): string {
  const params: string[] = [];
  if (since) {
    params.push(`since=${since}`);
  }
  if (appId) {
    params.push(`appId=${appId}`);
  }
  if (deploymentId) {
    // `contains:` is the requests-table filter syntax for a deployment id.
    params.push(`deploymentId=contains:${deploymentId}`);
  }
  const base = `${projectPath(scope)}/requests`;
  return params.length ? `${base}?${params.join("&")}` : base;
}

export function newAppPath({
  step,
  appId,
  ...scope
}: ProjectScope & { step?: string; appId?: string }): string {
  const params: string[] = [];
  if (step) {
    params.push(`step=${step}`);
  }
  if (appId) {
    params.push(`appId=${appId}`);
  }
  const base = `${projectPath(scope)}/apps/new`;
  return params.length ? `${base}?${params.join("&")}` : base;
}

export function appPath({ workspaceSlug, projectId, appId }: AppScope): string {
  return `${projectPath({ workspaceSlug, projectId })}/apps/${appId}`;
}

export function appSettingsPath(scope: AppScope): string {
  return `${appPath(scope)}/settings`;
}

export function appDeploymentsPath(scope: AppScope): string {
  return `${appPath(scope)}/deployments`;
}

export function deploymentPath({
  deploymentId,
  build,
  ...scope
}: AppScope & { deploymentId: string; build?: boolean }): string {
  const base = `${appDeploymentsPath(scope)}/${deploymentId}`;
  return build ? `${base}?build=true` : base;
}

export function openapiDiffPath({
  from,
  to,
  ...scope
}: AppScope & { from: string; to: string }): string {
  return `${appPath(scope)}/openapi-diff?from=${from}&to=${to}`;
}
