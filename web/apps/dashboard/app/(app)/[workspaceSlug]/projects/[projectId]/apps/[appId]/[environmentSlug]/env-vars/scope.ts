/**
 * Which environments the variables page reads and writes. App pages scope to
 * the route's environment; the new-app wizard has none yet and works across
 * all of them.
 */
export type EnvVarsScope = { kind: "environment"; environmentId: string } | { kind: "all" };

export function targetEnvironmentIds(scope: EnvVarsScope, allIds: string[]): string[] {
  return scope.kind === "all" ? allIds : [scope.environmentId];
}
