// Optimistic-insert filler; the server response overwrites these fields.
export const SERVER_PLACEHOLDER = "will-be-replaced-by-server";

export type ParsedFilter = { field: Array<string | number>; operator: string; value?: unknown };

/** Reads one string filter out of the parsed where clause of a load subset. */
export function extractStringFilter(
  filters: ParsedFilter[],
  fieldName: string,
  operator = "eq",
): string | undefined {
  const value = filters.find((f) => f.field.at(-1) === fieldName && f.operator === operator)?.value;
  return typeof value === "string" ? value : undefined;
}

export function extractStringValues(filters: ParsedFilter[], fieldName: string): string[] {
  const eqValue = extractStringFilter(filters, fieldName);
  if (eqValue !== undefined) {
    return [eqValue];
  }
  const inValue = filters.find((f) => f.field.at(-1) === fieldName && f.operator === "in")?.value;
  return Array.isArray(inValue)
    ? inValue.filter((v): v is string => typeof v === "string").sort()
    : [];
}
