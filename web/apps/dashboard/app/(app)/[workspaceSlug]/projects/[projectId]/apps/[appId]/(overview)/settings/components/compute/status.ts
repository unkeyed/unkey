import { match } from "@unkey/match";
import { type ComputeDraft, type EditResult, sameDraft } from "./draft";

export const UNAVAILABLE_FOR_SCHEDULING = "This region isn't accepting new instances right now.";

type AvailableRegion = { name: string; canSchedule: boolean };

export type AvailableRegions =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; regions: AvailableRegion[] };

export function availableFrom(query: {
  data: AvailableRegion[] | undefined;
  isError: boolean;
}): AvailableRegions {
  if (query.data) {
    return { status: "ready", regions: query.data };
  }
  return query.isError ? { status: "error" } : { status: "loading" };
}

export function unschedulableIn(available: AvailableRegions, names: string[]): string[] {
  if (available.status !== "ready") {
    return [];
  }
  const blocked = new Set(available.regions.filter((r) => !r.canSchedule).map((r) => r.name));
  return names.filter((name) => blocked.has(name));
}

export function addRegionBlocker(available: AvailableRegions, names: string[]): string | null {
  return match(available)
    .with({ status: "loading" }, () => "Loading regions…")
    .with({ status: "error" }, () => "Couldn't load regions. Reload the page to try again.")
    .with({ status: "ready" }, ({ regions }) =>
      regions.some((r) => r.canSchedule && !names.includes(r.name))
        ? null
        : "Your app already runs in every available region.",
    )
    .exhaustive();
}

export type CardStatus =
  | { type: "invalid"; reason: "unpicked" | "missing" }
  | { type: "taken" }
  | { type: "blocked"; names: string[] }
  | { type: "dirty"; draft: ComputeDraft }
  | { type: "clean" };

export function cardStatus(
  result: EditResult,
  available: AvailableRegions,
  base: ComputeDraft,
): CardStatus {
  if (!result.ok) {
    return result.reason === "taken" ? { type: "taken" } : { type: "invalid", reason: "unpicked" };
  }
  if (result.regionMissing) {
    return { type: "invalid", reason: "missing" };
  }
  const saved = new Set(base.regions.map((r) => r.name));
  const added = result.draft.regions.map((r) => r.name).filter((name) => !saved.has(name));
  const blocked = unschedulableIn(available, added);
  if (blocked.length > 0) {
    return { type: "blocked", names: blocked };
  }
  return sameDraft(result.draft, base) ? { type: "clean" } : { type: "dirty", draft: result.draft };
}
