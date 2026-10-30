import { describe, expect, it } from "vitest";
import { type ChartIncident, placeIncidents } from "./incident-markers";

const incident = (id: string, fromMs: number, toMs: number): ChartIncident => ({
  id,
  fromMs,
  toMs,
  label: id,
  content: null,
});

const combine = (ids: string[]) => incident(ids.join("+"), 1_100, 1_200);

const placed = (items: ChartIncident[], plotWidthPx = 400, activeId: string | null = null) =>
  placeIncidents({ items, combine }, activeId, [0, 1_000], plotWidthPx).map((p) => ({
    id: p.incident.id,
    leftPct: p.leftPct,
    widthPct: p.widthPct,
    range: p.range,
    active: p.active,
  }));

describe("placeIncidents", () => {
  it("places an incident as a percent of the domain", () => {
    expect(placed([incident("a", 200, 500)], 400, "a")).toEqual([
      { id: "a", leftPct: 20, widthPct: 30, range: [200, 500], active: true },
    ]);
  });

  it("widens a short incident to 4px around its centre", () => {
    expect(placed([incident("a", 500, 500)])).toEqual([
      { id: "a", leftPct: 49.5, widthPct: 1, range: [495, 505], active: false },
    ]);
  });

  it("keeps a widened incident inside the plot at either edge", () => {
    expect(placed([incident("start", 0, 0), incident("end", 1_000, 1_000)])).toEqual([
      { id: "start", leftPct: 0, widthPct: 1, range: [0, 10], active: false },
      { id: "end", leftPct: 99, widthPct: 1, range: [990, 1_000], active: false },
    ]);
  });

  it("fills a plot narrower than the minimum width without overflowing it", () => {
    expect(placed([incident("a", 500, 500)], 2)).toEqual([
      { id: "a", leftPct: 0, widthPct: 100, range: [0, 1_000], active: false },
    ]);
  });

  it("clamps an incident in the last bucket, past the domain end, to the right edge", () => {
    expect(placed([incident("late", 1_040, 1_060)])).toEqual([
      { id: "late", leftPct: 99, widthPct: 1, range: [990, 1_000], active: false },
    ]);
  });

  it("merges every incident past the domain end into one incident on the right edge", () => {
    expect(
      placed(
        [incident("a", 200, 500), incident("late1", 1_040, 1_060), incident("late2", 1_100, 1_300)],
        400,
        "late1+late2",
      ),
    ).toEqual([
      { id: "a", leftPct: 20, widthPct: 30, range: [200, 500], active: false },
      { id: "late1+late2", leftPct: 99, widthPct: 1, range: [990, 1_000], active: true },
    ]);
  });

  it("drops incidents before the domain and places nothing without a plot", () => {
    expect(placed([incident("before", -50, -10)])).toEqual([]);
    expect(placed([incident("a", 200, 500)], 0)).toEqual([]);
  });
});
