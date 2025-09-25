import { iconsPerField } from "@/components/logs/queries/utils";
import type { QuerySearchParams } from "@/lib/schemas/logs.filter.schema";
import { ChartActivity2 } from "@unkey/icons";
import { format } from "date-fns";
import React from "react";

export function formatFilterValues(
  filters: QuerySearchParams,
): Record<string, { operator: string; values: { value: string; color: string | null }[] }> {
  const transform = (field: string, value: string): { color: string | null; value: string } => {
    switch (field) {
      case "status":
        return {
          value:
            value === "200" ? "2xx" : value === "400" ? "4xx" : value === "500" ? "5xx" : value,
          color: value.startsWith("2")
            ? "bg-success-9"
            : value.startsWith("4")
              ? "bg-warning-9"
              : value.startsWith("5")
                ? "bg-error-9"
                : null,
        };
      case "methods":
        return { value: value.toUpperCase(), color: null };
      case "startTime":
      case "endTime":
        return { value: format(Number(value), "MMM d HH:mm:ss"), color: null };
      case "since":
        return { value: value, color: null };
      case "host":
      case "requestId":
      case "paths":
        return { value: value, color: null };
      default:
        return { value: value, color: null };
    }
  };

  const transformed: Record<
    string,
    { operator: string; values: { value: string; color: string | null }[] }
  > = {};

  // Handle special cases for different field types const transformed: Record<string, { operator: string; values: { value: string; color: string | null }[], icon: React.ReactNode }> = {};
  if (filters.startTime && filters.endTime) {
    transformed.time = {
      operator: "between",
      values: [
        transform("startTime", filters.startTime.toString()),
        transform("endTime", filters.endTime.toString()),
      ],
    };
  } else if (filters.startTime) {
    transformed.time = {
      operator: "starts from",
      values: [transform("startTime", filters.startTime.toString())],
    };
  } else if (filters.since) {
    transformed.time = {
      operator: "since",
      values: [{ value: filters.since, color: null }],
    };
  }

  Object.entries(filters).forEach(([field, value]) => {
    if (field === "startTime" || field === "endTime" || field === "since" || field === "time") {
      return [];
    }

    if (value === null) {
      return;
    }

    if (Array.isArray(value)) {
      transformed[field] = {
        operator: value[0]?.operator || "is",
        values: value.map((v) => transform(field, v.value.toString())),
      };
    } else {
      transformed[field] = {
        operator: "is",
        values: [transform(field, value.toString())],
      };
    }
  });

  return transformed;
}

export function getFilterFieldIcon(field: string): JSX.Element {
  const Icon = iconsPerField[field] || ChartActivity2;
  return React.createElement(Icon, {
    size: "md-regular",
    className: "justify-center",
  });
}

export const FieldsToTruncate = [
  "users",
  "workspaceId",
  "keyId",
  "apiId",
  "requestId",
  "responseId",
] as const;

export function shouldTruncateRow(field: string): boolean {
  return FieldsToTruncate.includes(field as (typeof FieldsToTruncate)[number]);
}
