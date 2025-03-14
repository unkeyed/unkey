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

// export const handleQueryKeyboard = (
//   e: React.KeyboardEvent,
//   containerRef: React.RefObject<HTMLElement>,
//   focusedTabIndex: number,
//   setFocusedTabIndex: React.Dispatch<React.SetStateAction<number>>,
//   selectedQueryIndex: number,
//   setSelectedQueryIndex: React.Dispatch<React.SetStateAction<number>>,
//   filterGroups: any[],
//   savedGroups: any[],
//   handleSelectedQuery: (index: number) => void,
//   setOpen: React.Dispatch<React.SetStateAction<boolean>>
// ) => {
//     // Adjust scroll speed as needed

//     if (containerRef.current) {
//       const scrollSpeed = 50;
//       // Handle up/down navigation
//       if (e.key === "ArrowUp" || e.key === "k" || e.key === "K") {
//         e.preventDefault();

//         const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
//         const totalItems = currentList.length - 1;
//         containerRef.current.scrollTop -= scrollSpeed;
//         if (totalItems === 0) {
//           return;
//         }

//         // Move selection up, wrap to bottom if at top
//         setSelectedQueryIndex((prevIndex) => (prevIndex > 0 ? prevIndex - 1 : 0));
//       } else if (e.key === "ArrowDown" || e.key === "j" || e.key === "J") {
//         e.preventDefault();

//         containerRef.current.scrollTop += scrollSpeed;
//         const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
//         const totalItems = currentList.length - 1;

//         if (totalItems === 0) {
//           return;
//         }

//         // Move selection down, wrap to top if at bottom
//         setSelectedQueryIndex((prevIndex) =>
//           prevIndex < totalItems - 1 ? prevIndex + 1 : totalItems,
//         );
//       }
//     }
//     // Handle tab navigation
//     if (e.key === "ArrowLeft" || e.key === "h" || e.key === "H") {
//       // Move to All tab

//           // Adjust scroll speed as needed

//           // Rest of the function remains the same
//           setFocusedTabIndex(0);
//       setSelectedQueryIndex(0);
//     } else if (e.key === "ArrowRight" || e.key === "l" || e.key === "L") {
//       // Move to Saved tab
//       setFocusedTabIndex(1);
//       setSelectedQueryIndex(0);
//     } else if (e.key === "Enter" || e.key === " ") {
//       // Apply the selected filter
//       const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
//       if (currentList.length > 0 && selectedQueryIndex < currentList.length) {
//         handleSelectedQuery(selectedQueryIndex);
//         setOpen(false);
//       }
//     }
//   };

// export const getIcon = ({ field }: { field: LogsFilterField | AuditLogsFilterField | RatelimitFilterField }) => {
//   // Get the appropriate icon based on the field name
//   switch (field.toLowerCase()) {
//     case "status":
//       return ChartActivity2;
//     case "time":
//     case "since":
//     case "date":
//       return Clock;
//     case "tag":
//       return Tag;
//     case "user":
//       return User;
//     case "layer":
//     case "layers":
//       return Layers2;
//     case "success":
//     case "verified":
//       return CircleCheck;
//     case "conversion":
//     case "convert":
//       return Conversion;
//     case "bookmark":
//       return Bookmark;
//     case "link":
//       return Link4;
//     case "calendar":
//       return Calendar;
//     default:
//       return ChartActivity2; // Default icon
//   }
// };
