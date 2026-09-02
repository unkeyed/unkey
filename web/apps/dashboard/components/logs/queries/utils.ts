import {
  differenceInDays,
  differenceInHours,
  differenceInMinutes,
  differenceInMonths,
  differenceInSeconds,
  differenceInWeeks,
  differenceInYears,
  format,
} from "date-fns";
import React from "react";

import { auditLogsFilterFieldEnum } from "@/app/(app)/[workspaceSlug]/audit/filters.schema";
import { ratelimitFilterFieldEnum } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/logs/filters.schema";
import type { IconProps } from "nucleo-ui-outline-18";
import {
  IconBucketOutline18,
  IconCalendarEventOutline18,
  IconChartActivity2Outline18,
  IconClockOutline18,
  IconConversionOutline18,
  IconFingerprintOutline18,
  IconFocusOutline18,
  IconFolderCloudOutline18,
  IconKeyOutline18,
  IconLink4Outline18,
  IconUserSearchOutline18,
} from "nucleo-ui-outline-18";

import type { AuditLogsFilterField } from "@/app/(app)/[workspaceSlug]/audit/filters.schema";
import type { RatelimitFilterField } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/logs/filters.schema";
import { namespaceListFilterFieldEnum } from "@/app/(app)/[workspaceSlug]/ratelimits/_components/namespace-list-filters.schema";
import {
  type LogsFilterField,
  type QuerySearchParams,
  logsFilterFieldEnum,
} from "@/lib/schemas/logs.filter.schema";
import type { FC } from "react";

export const iconsPerField: Record<string, FC<IconProps>> = {
  status: IconChartActivity2Outline18,
  methods: IconConversionOutline18,
  paths: IconLink4Outline18,
  time: IconClockOutline18,
  startTime: IconClockOutline18,
  endTime: IconClockOutline18,
  since: IconClockOutline18,
  bucket: IconBucketOutline18,
  events: IconCalendarEventOutline18,
  users: IconUserSearchOutline18,
  rootKeys: IconKeyOutline18,
  host: IconFolderCloudOutline18,
  requestId: IconFingerprintOutline18,
  identifiers: IconFocusOutline18,
  requestIds: IconFingerprintOutline18,
};

export function parseValue(value: string) {
  // Check if value can be parsed as a number
  if (value === "passed") {
    return { color: "bg-success-9", phrase: value };
  }
  if (value === "blocked") {
    return { color: "bg-warning-9", phrase: value };
  }
  const isNumeric = !Number.isNaN(Number.parseFloat(value)) && Number.isFinite(Number(value));
  if (!isNumeric) {
    return { color: null, phrase: value };
  }
  const numValue = Number(value);
  if (numValue >= 200 && numValue < 300) {
    return { color: "bg-success-9", phrase: value === "200" ? "2xx" : value };
  }
  if (numValue >= 400 && numValue < 500) {
    return { color: "bg-warning-9", phrase: value === "400" ? "4xx" : value };
  }
  if (numValue >= 500 && numValue < 600) {
    return { color: "bg-error-9", phrase: value === "500" ? "5xx" : value };
  }

  return { color: null, phrase: value };
}

export function formatFilterValue(
  filters: QuerySearchParams,
): Record<string, { operator: string; values: { value: string; color: string | null }[] }> {
  // Handle special cases for different field types
  const transform = (field: string, value: string): { color: string | null; value: string } => {
    switch (field) {
      case "status":
        return {
          value:
            value === "200" ? "2xx" : value === "400" ? "4xx" : value === "500" ? "5xx" : value,
          color:
            value === "200"
              ? "bg-success-9"
              : value === "400"
                ? "bg-warning-9"
                : value === "500"
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

export function getFilterFieldIcon(field: string): React.ReactElement {
  const Icon = iconsPerField[field] || IconChartActivity2Outline18;
  return React.createElement(Icon, {
    className: "size-4 justify-center",
  });
}

export const getFilterFieldEnum = () => {
  // Combine all unique filter field values from different enums
  const filterFieldList: { field: string; icon: FC<IconProps> }[] = [];
  for (const field of Object.values(logsFilterFieldEnum)) {
    filterFieldList.push(field);
  }
  for (const field of Object.values(auditLogsFilterFieldEnum)) {
    filterFieldList.push(field);
  }
  for (const field of Object.values(ratelimitFilterFieldEnum)) {
    filterFieldList.push(field);
  }
  for (const field of Object.values(namespaceListFilterFieldEnum)) {
    filterFieldList.push(field);
  }
  return filterFieldList;
};

export const getSinceTime = (date: number) => {
  const now = new Date();
  const seconds = differenceInSeconds(now, date);
  if (seconds < 60) {
    return "just now";
  }
  const minutes = differenceInMinutes(now, date);
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = differenceInHours(now, date);
  if (hours < 24) {
    return `${hours}h ago`;
  }
  const days = differenceInDays(now, date);
  if (days < 7) {
    return `${days}d ago`;
  }

  const weeks = differenceInWeeks(now, date);
  if (weeks < 4) {
    return `${weeks}w ago`;
  }

  const months = differenceInMonths(now, date);
  if (months < 12) {
    return `${months} month(s) ago`;
  }

  const years = differenceInYears(now, date);
  return `${years} year(s) ago`;
};

export type FullFilterField = LogsFilterField | AuditLogsFilterField | RatelimitFilterField;

export const FieldsToTruncate = [
  "paths",
  "methods",
  "events",
  "identifiers",
  "requestIds",
  "rootKeys",
  "users",
  "bucket",
  "host",
  "requestId",
];
