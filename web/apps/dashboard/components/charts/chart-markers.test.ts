import { describe, expect, it } from "vitest";
import { activeMarkerSpan, placeMarkers } from "./chart-markers";

const items = [
  { id: "before", atMs: 500, span: { endMs: 1_500 } },
  { id: "inside", atMs: 1_500, span: { endMs: 1_800 } },
  { id: "last", atMs: 1_800, span: { endMs: null } },
  { id: "unspanned", atMs: 1_900, span: null },
];

describe("placeMarkers", () => {
  it("places markers inside the domain as a percent of its width", () => {
    expect(placeMarkers(items, [1_000, 2_000])).toEqual([
      {
        key: "inside",
        ids: ["inside"],
        fromMs: 1_500,
        toMs: 1_500,
        x: 1_500,
        leftPct: 50,
        title: "Deploy",
      },
      {
        key: "last",
        ids: ["last"],
        fromMs: 1_800,
        toMs: 1_800,
        x: 1_800,
        leftPct: 80,
        title: "Deploy",
      },
      {
        key: "unspanned",
        ids: ["unspanned"],
        fromMs: 1_900,
        toMs: 1_900,
        x: 1_900,
        leftPct: 90,
        title: "Deploy",
      },
    ]);
  });

  it("keeps markers close together as separate markers", () => {
    const close = [
      { id: "a", atMs: 1_000, span: null },
      { id: "b", atMs: 1_001, span: null },
    ];
    expect(placeMarkers(close, [1_000, 2_000]).map((m) => [m.key, m.ids])).toEqual([
      ["a", ["a"]],
      ["b", ["b"]],
    ]);
  });

  it("groups markers past the domain end into one marker on the right edge", () => {
    expect(placeMarkers(items, [1_250, 1_750])).toEqual([
      {
        key: "inside",
        ids: ["inside"],
        fromMs: 1_500,
        toMs: 1_500,
        x: 1_500,
        leftPct: 50,
        title: "Deploy",
      },
      {
        key: "late",
        ids: ["last", "unspanned"],
        fromMs: 1_800,
        toMs: 1_900,
        x: 1_750,
        leftPct: 100,
        title: "2 deploys",
      },
    ]);
  });

  it("places nothing on an empty domain", () => {
    expect(placeMarkers(items, [1_000, 1_000])).toEqual([]);
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
