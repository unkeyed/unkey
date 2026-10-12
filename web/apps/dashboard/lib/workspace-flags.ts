export type WorkspaceFlag = {
  readonly slug: string;
  readonly defaultValue: boolean;
};

export function useFlag({
  workspace,
  flag,
}: {
  workspace: { flags: Partial<Record<string, boolean>> } | null | undefined;
  flag: WorkspaceFlag;
}): boolean {
  const value = workspace?.flags[flag.slug];
  return typeof value === "boolean" ? value : flag.defaultValue;
}
