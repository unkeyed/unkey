import { describe, expect, it } from "vitest";
import { activeMarkerSpan, markerSlots, placeMarkers, stackHeightPx } from "./chart-markers";

const items = [
  { id: "before", atMs: 500, span: { endMs: 1_500 }, shape: "dot" as const, label: null },
  { id: "inside", atMs: 1_500, span: { endMs: 1_800 }, shape: "dot" as const, label: null },
  { id: "last", atMs: 1_800, span: { endMs: null }, shape: "diamond" as const, label: null },
  { id: "unspanned", atMs: 1_900, span: null, shape: "diamond" as const, label: null },
];

describe("placeMarkers", () => {
  it("places markers inside the domain as a percent of its width", () => {
    expect(placeMarkers(items, [1_000, 2_000], 0)).toEqual([
      {
        key: "inside",
        ids: ["inside"],
        atMs: 1_500,
        x: 1_500,
        leftPct: 50,
        late: false,
        shape: "dot",
        label: null,
      },
      {
        key: "last",
        ids: ["last"],
        atMs: 1_800,
        x: 1_800,
        leftPct: 80,
        late: false,
        shape: "diamond",
        label: null,
      },
      {
        key: "unspanned",
        ids: ["unspanned"],
        atMs: 1_900,
        x: 1_900,
        leftPct: 90,
        late: false,
        shape: "diamond",
        label: null,
      },
    ]);
  });

  it("groups markers past the domain end into one marker on the right edge", () => {
    expect(placeMarkers(items, [1_250, 1_750], 0)).toEqual([
      {
        key: "inside",
        ids: ["inside"],
        atMs: 1_500,
        x: 1_500,
        leftPct: 50,
        late: false,
        shape: "dot",
        label: null,
      },
      {
        key: "late",
        ids: ["last", "unspanned"],
        atMs: 1_900,
        x: 1_750,
        leftPct: 100,
        late: true,
        shape: "diamond",
        label: null,
      },
    ]);
  });

  it("places nothing on an empty domain", () => {
    expect(placeMarkers(items, [1_000, 1_000], 0)).toEqual([]);
  });
});

describe("placeMarkers stacking", () => {
  const diamond = (id: string, atMs: number) => ({
    id,
    atMs,
    span: null,
    shape: "diamond" as const,
    label: `Crash ${id}`,
  });

  it("stacks diamonds closer than the gap and keeps dots and distant diamonds apart", () => {
    const placed = placeMarkers(
      [
        diamond("a", 1_000),
        diamond("b", 1_040),
        { id: "deploy", atMs: 1_050, span: null, shape: "dot" as const, label: null },
        diamond("c", 1_060),
        diamond("d", 1_500),
      ],
      [1_000, 2_000],
      5,
    );
    expect(placed.map((m) => [m.key, m.ids, m.shape, m.label])).toEqual([
      ["a", ["a", "b"], "diamond", null],
      ["deploy", ["deploy"], "dot", null],
      ["c", ["c"], "diamond", "Crash c"],
      ["d", ["d"], "diamond", "Crash d"],
    ]);
  });
});

describe("markerSlots", () => {
  const diamonds = ["a", "b", "c", "d", "e", "f"].map((id, i) => ({
    id,
    atMs: 1_000 + i,
    span: null,
    shape: "diamond" as const,
    label: `Crash ${id}`,
  }));
  const labels = new Map(diamonds.map((d) => [d.id, d.label]));

  it("stacks up to four diamonds, one per slot, then one overflow slot", () => {
    const [stack] = placeMarkers(diamonds, [1_000, 2_000], 5);
    const slots = markerSlots(stack, labels);
    expect(slots.map((s) => [s.key, s.ids, s.layer, s.overflow])).toEqual([
      ["a", ["a"], 0, 0],
      ["b", ["b"], 1, 0],
      ["c", ["c"], 2, 0],
      ["d", ["d"], 3, 0],
      ["a:more", ["e", "f"], 4, 2],
    ]);
    expect(slots[1].label.startsWith("Crash b at ")).toBe(true);
    expect(stackHeightPx([stack])).toBe(40);
  });

  it("keeps a single marker in one slot with no extra height", () => {
    const placed = placeMarkers(diamonds.slice(0, 1), [1_000, 2_000], 5);
    expect(markerSlots(placed[0], labels).map((s) => [s.ids, s.layer])).toEqual([[["a"], 0]]);
    expect(stackHeightPx(placed)).toBe(0);
  });
});

describe("activeMarkerSpan", () => {
  it("runs an open span to the domain end", () => {
    expect(activeMarkerSpan(items, "last", [1_000, 2_000])).toEqual([1_800, 2_000]);
  });

  it("clamps a span that starts before the domain", () => {
    expect(activeMarkerSpan(items, "before", [1_000, 2_000])).toEqual([1_000, 1_500]);
  });

  it("shades nothing for a marker without a span", () => {
    expect(activeMarkerSpan(items, "unspanned", [1_000, 2_000])).toBeNull();
  });

  it("returns null with no active marker", () => {
    expect(activeMarkerSpan(items, null, [1_000, 2_000])).toBeNull();
  });
});
