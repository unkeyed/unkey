import {
  differenceInDays,
  differenceInHours,
  differenceInMinutes,
  differenceInMonths,
  differenceInSeconds,
  differenceInWeeks,
  differenceInYears,
} from "date-fns";

import { auditLogsFilterFieldEnum } from "@/app/(app)/audit/filters.schema";
import { logsFilterFieldEnum } from "@/app/(app)/logs/filters.schema";
import { ratelimitFilterFieldEnum } from "@/app/(app)/ratelimits/[namespaceId]/logs/filters.schema";
import { ratelimitListFilterFieldEnum } from "@/app/(app)/ratelimits/_components/filters.schema";
import {
  Bucket,
  CalendarEvent,
  ChartActivity2,
  Clock,
  Conversion,
  Fingerprint,
  Focus,
  FolderCloud,
  Key,
  Link4,
  UserSearch,
} from "@unkey/icons";

import type { AuditLogsFilterField } from "@/app/(app)/audit/filters.schema";
import type { LogsFilterField } from "@/app/(app)/logs/filters.schema";
import type { RatelimitFilterField } from "@/app/(app)/ratelimits/[namespaceId]/logs/filters.schema";
import type { IconProps } from "@unkey/icons/src/props";
import type { FC } from "react";

export const iconsPerField: Record<string, FC<IconProps>> = {
  status: ChartActivity2,
  methods: Conversion,
  path: Link4,
  time: Clock,
  startTime: Clock,
  endTime: Clock,
  since: Clock,
  bucket: Bucket,
  events: CalendarEvent,
  users: UserSearch,
  rootKeys: Key,
  host: FolderCloud,
  requestId: Fingerprint,
  identifiers: Focus,
  requestIds: Fingerprint,
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
  for (const field of Object.values(ratelimitListFilterFieldEnum)) {
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
