import { describe, expect, it } from "vitest";
import { dropHiddenBucketFilters } from "./utils";

describe("dropHiddenBucketFilters", () => {
  it("keeps dashboard bucket filters", () => {
    const parsed = {
      filters: [
        {
          field: "bucket" as const,
          filters: [{ operator: "is" as const, value: "unkey_mutations" }],
        },
      ],
    };
    expect(dropHiddenBucketFilters(parsed)).toEqual(parsed);
  });

  it("drops bucket filters that name a hidden bucket", () => {
    const parsed = {
      filters: [
        {
          field: "bucket" as const,
          filters: [{ operator: "is" as const, value: "unkey_backoffice" }],
        },
        { field: "events" as const, filters: [{ operator: "is" as const, value: "key.create" }] },
      ],
    };
    expect(dropHiddenBucketFilters(parsed)).toEqual({
      filters: [{ field: "events", filters: [{ operator: "is", value: "key.create" }] }],
    });
  });

  it("drops bucket filters with non string values", () => {
    const parsed = {
      filters: [{ field: "bucket" as const, filters: [{ operator: "is" as const, value: 42 }] }],
    };
    expect(dropHiddenBucketFilters(parsed)).toEqual({ filters: [] });
  });
});
