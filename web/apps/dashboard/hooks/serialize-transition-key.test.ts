import { describe, expect, it } from "vitest";
import { serializeFilters, serializeSorts } from "./serialize-transition-key";

// These helpers produce the transitionKey usePageTransition compares across
// renders to decide when to reset pagination. The load-bearing guarantee is
// content-stability: equal filter/sort content must yield equal keys, and any
// meaningful difference must yield a different key. A regression here would
// silently break (or spuriously trigger) page resets across every table.
describe("serializeFilters", () => {
  it("produces the same key for equal content in a fresh array", () => {
    const a = serializeFilters([{ field: "status", operator: "is", value: "blocked" }]);
    const b = serializeFilters([{ field: "status", operator: "is", value: "blocked" }]);

    expect(a).toBe(b);
  });

  it("produces a different key when any component changes", () => {
    const base = serializeFilters([{ field: "status", operator: "is", value: "blocked" }]);

    expect(serializeFilters([{ field: "status", operator: "is", value: "passed" }])).not.toBe(base);
    expect(serializeFilters([{ field: "status", operator: "not", value: "blocked" }])).not.toBe(
      base,
    );
    expect(serializeFilters([{ field: "outcome", operator: "is", value: "blocked" }])).not.toBe(
      base,
    );
  });

  // The reason this uses JSON tuples instead of `field:op:value` joined by `|`:
  // a value containing the delimiter must not let two distinct states collapse
  // to the same key, which would suppress a real page reset.
  it("does not collide when a value contains delimiter characters", () => {
    const oneFilter = serializeFilters([{ field: "name", operator: "is", value: "a|b" }]);
    const twoFilters = serializeFilters([
      { field: "name", operator: "is", value: "a" },
      { field: "b", operator: "is", value: "" },
    ]);
    expect(oneFilter).not.toBe(twoFilters);

    const colonValue = serializeFilters([{ field: "name", operator: "is", value: "a:b" }]);
    const colonSplit = serializeFilters([{ field: "name", operator: "is:a", value: "b" }]);
    expect(colonValue).not.toBe(colonSplit);
  });

  // Filter values are not always strings (timestamps, counts); encoding must be
  // deterministic so the key stays stable for equal values.
  it("encodes non-string values deterministically", () => {
    const a = serializeFilters([{ field: "startTime", operator: "is", value: 1700000000 }]);
    const b = serializeFilters([{ field: "startTime", operator: "is", value: 1700000000 }]);
    expect(a).toBe(b);
    // A number and its string form are distinct states and must differ.
    expect(
      serializeFilters([{ field: "startTime", operator: "is", value: "1700000000" }]),
    ).not.toBe(a);
  });

  it("returns a stable key for no filters", () => {
    expect(serializeFilters([])).toBe(serializeFilters([]));
  });
});

describe("serializeSorts", () => {
  it("distinguishes column and direction changes", () => {
    const base = serializeSorts([{ column: "createdAt", direction: "desc" }]);
    expect(serializeSorts([{ column: "createdAt", direction: "asc" }])).not.toBe(base);
    expect(serializeSorts([{ column: "name", direction: "desc" }])).not.toBe(base);
  });

  it("preserves order (multi-column sort is order-sensitive)", () => {
    const ab = serializeSorts([
      { column: "createdAt", direction: "desc" },
      { column: "name", direction: "asc" },
    ]);
    const ba = serializeSorts([
      { column: "name", direction: "asc" },
      { column: "createdAt", direction: "desc" },
    ]);
    expect(ab).not.toBe(ba);
  });

  it("returns a stable key for no sorts", () => {
    expect(serializeSorts([])).toBe(serializeSorts([]));
  });
});
