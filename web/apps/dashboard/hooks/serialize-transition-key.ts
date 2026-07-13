// Helpers for building the stable `transitionKey` that usePageTransition
// watches. The key must be a plain string derived from filter/sort *content*
// (not array identity) so that a new array reference for the same values does
// not read as a transition. Each hook composes these with its own extra
// dimensions (time window, search) around them — see usePageTransition.
//
// Encoding is JSON tuples rather than delimiter-joined text so the result is
// unambiguous: a filter value containing a separator character cannot make two
// distinct states collapse to the same key (which would suppress a real page
// reset). The strings are only ever compared for equality, never parsed.

type KeyableFilter = { field: string; operator: string; value: unknown };
type KeyableSort = { column: string; direction: string };

export function serializeFilters(filters: readonly KeyableFilter[]): string {
  return JSON.stringify(filters.map((f) => [f.field, f.operator, f.value]));
}

export function serializeSorts(sorts: readonly KeyableSort[]): string {
  return JSON.stringify(sorts.map((s) => [s.column, s.direction]));
}
