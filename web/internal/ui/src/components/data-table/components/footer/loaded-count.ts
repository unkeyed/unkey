import { formatCompactNumber } from "../../../../lib/format";

export function loadedCountLabel({
  loadedCount,
  totalCount,
  singular,
  plural,
  scope,
}: {
  loadedCount: number;
  totalCount: number | null;
  singular: string;
  plural: string;
  scope?: string;
}) {
  const noun = (totalCount ?? loadedCount) === 1 ? singular : plural;
  const loaded = formatCompactNumber(loadedCount);
  if (totalCount !== null) {
    return `${loaded} of ${formatCompactNumber(totalCount)} ${noun}`;
  }
  return scope ? `${loaded} ${noun} ${scope}` : `${loaded} ${noun}`;
}
