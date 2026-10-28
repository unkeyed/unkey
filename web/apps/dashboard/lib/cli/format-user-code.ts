export function formatUserCode(code: string): string {
  const symbols = code.trim().toUpperCase().replaceAll("-", "");
  return `${symbols.slice(0, 4)}-${symbols.slice(4)}`;
}
