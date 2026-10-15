export const ALL_ENVIRONMENTS = "all";

export function targetEnvironmentIds(
  environmentId: string,
  environmentIds: readonly string[],
): readonly string[] {
  return environmentId === ALL_ENVIRONMENTS ? environmentIds : [environmentId];
}
