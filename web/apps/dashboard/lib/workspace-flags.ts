import type { WorkspaceFlag } from "./flags";

export function resolveFlag({
  workspace,
  flag,
}: {
  workspace: { flags: Partial<Record<string, boolean>> } | null | undefined;
  flag: WorkspaceFlag;
}): boolean {
  const value = workspace?.flags[flag.slug];
  return typeof value === "boolean" ? value : flag.defaultValue;
}
