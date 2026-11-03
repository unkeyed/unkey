"use client";
import { useState } from "react";

export type StripeSlot = { live: boolean; position: number };

export function stripeIndexOf(slot: StripeSlot, live: { count: number; shift: number }): number {
  return slot.live ? live.count - 1 - slot.position + live.shift : slot.position;
}

function liveShiftAfter(
  previous: { ids: readonly string[]; shift: number },
  ids: readonly string[],
): number {
  if (ids.length === 0) {
    return 0;
  }
  const kept = new Set(ids);
  const evicted = previous.ids.filter((id) => !kept.has(id)).length;
  return (previous.shift + evicted) % 2;
}

export function useLiveStripeShift<Row>(rows: readonly Row[], idOf: (row: Row) => string): number {
  const ids = rows.map(idOf);
  const [seen, setSeen] = useState<{ ids: readonly string[]; shift: number }>({ ids, shift: 0 });
  if (!sameIds(seen.ids, ids)) {
    setSeen({ ids, shift: liveShiftAfter(seen, ids) });
  }
  return seen.shift;
}

function sameIds(a: readonly string[], b: readonly string[]) {
  return a.length === b.length && a.every((id, index) => id === b[index]);
}
