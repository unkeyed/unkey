import { describe, expect, it } from "vitest";
import {
  type FilterDimension,
  type FilterOption,
  searchHits,
  splitPanelOptions,
} from "./filter-model";

function dimension(options: FilterOption[], extra: Partial<FilterDimension> = {}): FilterDimension {
  return {
    id: "outcomes",
    label: "Outcome",
    groupLabel: "Outcomes",
    hideEmpty: true,
    options,
    selected: [],
    onToggle: () => {},
    onClear: () => {},
    ...extra,
  };
}

const valid = { value: "valid", label: "Valid", count: 3 };
const expired = { value: "expired", label: "Expired", count: 0 };

describe("splitPanelOptions", () => {
  it("keeps an applied option visible even with no requests", () => {
    const panel = dimension([valid, expired], { selected: ["expired"] });
    const { visible, empty } = splitPanelOptions(panel, "", false);

    expect(visible.map((option) => option.value)).toEqual(["valid", "expired"]);
    expect(empty).toEqual([]);
  });

  it("shows every option when the dimension does not hide empty ones", () => {
    const panel = dimension([valid, expired], { hideEmpty: false });
    const { visible, empty } = splitPanelOptions(panel, "", false);

    expect(visible.map((option) => option.value)).toEqual(["valid", "expired"]);
    expect(empty).toEqual([]);
  });

  it("hides empty options until they are revealed", () => {
    const panel = dimension([valid, expired]);

    expect(splitPanelOptions(panel, "", false).visible.map((o) => o.value)).toEqual(["valid"]);
    expect(splitPanelOptions(panel, "", false).empty.map((o) => o.value)).toEqual(["expired"]);
    expect(splitPanelOptions(panel, "", true).visible.map((o) => o.value)).toEqual([
      "valid",
      "expired",
    ]);
  });
});

describe("searchHits", () => {
  const dimensions = [
    dimension([valid, expired]),
    dimension([{ value: "k1", label: "Ingest", search: "key_abc" }], {
      id: "keys",
      groupLabel: "Keys",
    }),
  ];

  it("groups the matches under their own dimension", () => {
    const hits = searchHits(dimensions, "val");

    expect(hits.map((hit) => hit.options.map((option) => option.value))).toEqual([["valid"], []]);
  });

  it("matches a key on text the row never shows", () => {
    const hits = searchHits(dimensions, "key_abc");

    expect(hits.map((hit) => hit.options.map((option) => option.value))).toEqual([[], ["k1"]]);
  });

  it("returns no options at all when nothing matches", () => {
    const hits = searchHits(dimensions, "nothing");

    expect(hits.every((hit) => hit.options.length === 0)).toBe(true);
  });
});
