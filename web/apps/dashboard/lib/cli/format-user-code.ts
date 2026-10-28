export function formatUserCode(code: string): string {
  return [...code.trim().toUpperCase()].join(" ");
}
