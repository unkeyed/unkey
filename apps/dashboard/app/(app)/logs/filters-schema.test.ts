import { describe, expect, it, vi } from "vitest";
import {
  filterFieldConfig,
  transformStructuredOutputToFilters,
  validateFieldValue,
} from "./filters.schema";
import type { FilterValue } from "./filters.type";

vi.stubGlobal("crypto", {
  randomUUID: vi.fn(() => "test-uuid"),
});

describe("transformStructuredOutputToFilters", () => {
  it("should transform structured output to filters correctly", () => {
    const input = {
      filters: [
        {
          field: "status",
          filters: [{ operator: "is", value: 404 }],
        },
        {
          field: "paths",
          filters: [
            { operator: "contains", value: "api" },
            { operator: "startsWith", value: "/v1" },
          ],
        },
      ],
    };

    //@ts-ignore
    const result = transformStructuredOutputToFilters(input, undefined);

    expect(result).toHaveLength(3);
    expect(result[0]).toMatchObject({
      field: "status",
      operator: "is",
      value: 404,
      metadata: {
        colorClass: "bg-warning-8",
      },
    });
    expect(result[1]).toMatchObject({
      field: "paths",
      operator: "contains",
      value: "api",
    });
    expect(result[2]).toMatchObject({
      field: "paths",
      operator: "startsWith",
      value: "/v1",
    });
  });

  it("should deduplicate filters with existing filters", () => {
    const existingFilters: FilterValue[] = [
      {
        id: "123",
        field: "status",
        operator: "is",
        value: 404,
        metadata: { colorClass: "bg-warning-8" },
      },
    ];

    const input = {
      filters: [
        {
          field: "status",
          filters: [{ operator: "is", value: 404 }],
        },
        {
          field: "paths",
          filters: [{ operator: "contains", value: "api" }],
        },
      ],
    };

    //@ts-ignore
    const result = transformStructuredOutputToFilters(input, existingFilters);

    expect(result).toHaveLength(2);
    expect(result[1]).toMatchObject({
      field: "paths",
      operator: "contains",
      value: "api",
    });
  });
});

describe("validateFieldValue", () => {
  it("should validate status codes correctly", () => {
    expect(validateFieldValue("status", 200)).toBe(true);
    expect(validateFieldValue("status", 404)).toBe(true);
    expect(validateFieldValue("status", 500)).toBe(true);
    expect(validateFieldValue("status", 600)).toBe(false);
  });

  it("should validate HTTP methods correctly", () => {
    expect(validateFieldValue("methods", "GET")).toBe(true);
    expect(validateFieldValue("methods", "POST")).toBe(true);
    expect(validateFieldValue("methods", "INVALID")).toBe(false);
  });

  it("should validate string fields correctly", () => {
    expect(validateFieldValue("paths", "/api/v1")).toBe(true);
    expect(validateFieldValue("host", "example.com")).toBe(true);
    expect(validateFieldValue("requestId", "req-123")).toBe(true);
  });

  it("should validate number fields correctly", () => {
    expect(validateFieldValue("startTime", 1234567890)).toBe(true);
    expect(validateFieldValue("endTime", 1234567890)).toBe(true);
  });
});

describe("filterFieldConfig", () => {
  it("should have correct status color classes", () => {
    expect(filterFieldConfig.status.getColorClass!(200)).toBe("bg-success-9");
    expect(filterFieldConfig.status.getColorClass!(404)).toBe("bg-warning-8");
    expect(filterFieldConfig.status.getColorClass!(500)).toBe("bg-error-9");
  });

  it("should have correct operators for each field", () => {
    expect(filterFieldConfig.status.operators).toEqual(["is"]);
    expect(filterFieldConfig.paths.operators).toEqual(["is", "contains", "startsWith", "endsWith"]);
    expect(filterFieldConfig.host.operators).toEqual(["is"]);
    expect(filterFieldConfig.requestId.operators).toEqual(["is"]);
  });

  it("should have correct field types", () => {
    expect(filterFieldConfig.status.type).toBe("number");
    expect(filterFieldConfig.methods.type).toBe("string");
    expect(filterFieldConfig.paths.type).toBe("string");
    expect(filterFieldConfig.host.type).toBe("string");
  });
});
