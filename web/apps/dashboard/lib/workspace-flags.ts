export function useFlag({
  workspace,
  name,
  default: defaultValue,
}: {
  workspace: { flags: Partial<Record<string, boolean>> } | null | undefined;
  name: string;
  default: boolean;
}): boolean {
  const value = workspace?.flags[name];
  return typeof value === "boolean" ? value : defaultValue;
}
