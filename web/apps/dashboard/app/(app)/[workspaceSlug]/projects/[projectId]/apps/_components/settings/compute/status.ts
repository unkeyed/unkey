const UNAVAILABLE_FOR_SCHEDULING = "This region isn't accepting new instances right now.";

type AvailableRegion = { name: string; canSchedule: boolean };

type AvailableRegions =
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

const REGIONS_NOTE: Record<AvailableRegions["status"], string | undefined> = {
  loading: "Loading regions…",
  error: "Couldn't load regions. Reload the page to try again.",
  ready: undefined,
};

export function regionsNote(available: AvailableRegions): string | undefined {
  return REGIONS_NOTE[available.status];
}

type RegionState = "on" | "off" | "last" | "unavailable" | "degraded";

export const REGION_STATES: Record<
  RegionState,
  { selected: boolean; locked: boolean; warning: boolean; hint: string | undefined }
> = {
  on: { selected: true, locked: false, warning: false, hint: undefined },
  off: { selected: false, locked: false, warning: false, hint: undefined },
  last: {
    selected: true,
    locked: true,
    warning: false,
    hint: "Your app needs at least one region.",
  },
  unavailable: {
    selected: false,
    locked: true,
    warning: false,
    hint: UNAVAILABLE_FOR_SCHEDULING,
  },
  degraded: { selected: true, locked: false, warning: true, hint: UNAVAILABLE_FOR_SCHEDULING },
};

type RegionChoice = { name: string; state: RegionState };

function regionState(selected: string[], name: string, schedulable: boolean): RegionState {
  if (!selected.includes(name)) {
    return schedulable ? "off" : "unavailable";
  }
  if (selected.length === 1) {
    return "last";
  }
  return schedulable ? "on" : "degraded";
}

export function regionChoices(available: AvailableRegions, selected: string[]): RegionChoice[] {
  const listed = available.status === "ready" ? available.regions : [];
  const known = new Set(listed.map((r) => r.name));
  return [
    ...listed.map((r) => ({ name: r.name, state: regionState(selected, r.name, r.canSchedule) })),
    ...selected
      .filter((name) => !known.has(name))
      .map((name) => ({ name, state: regionState(selected, name, true) })),
  ];
}
