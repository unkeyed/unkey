import {
  Ban,
  CircleCaretRight,
  CircleCheck,
  CircleHalfDottedClock,
  ShieldKey,
  TriangleWarning2,
} from "@unkey/icons";

export type StatusType =
  | "disabled"
  | "low-credits"
  | "expires-soon"
  | "rate-limited"
  | "validation-issues"
  | "operational";

export interface StatusInfo {
  type: StatusType;
  label: string;
  color: string;
  icon: React.ReactNode;
  tooltip: string;
  priority: number; // Lower number = higher priority
}

export const STATUS_DEFINITIONS: Record<StatusType, StatusInfo> = {
  "low-credits": {
    type: "low-credits",
    label: "Low Credits",
    color: "bg-errorA-3 text-errorA-11",
    icon: <CircleCaretRight iconSize="sm-regular" />,
    tooltip: "This key has a low credit balance. Top it off to prevent disruptions.",
    priority: 1,
  },
  "rate-limited": {
    type: "rate-limited",
    label: "Ratelimited",
    color: "bg-errorA-3 text-errorA-11",
    icon: <TriangleWarning2 iconSize="sm-regular" />,
    tooltip:
      "This key is getting ratelimited frequently. Check the configured ratelimits and reach out to your user about their usage.",
    priority: 2,
  },
  "expires-soon": {
    type: "expires-soon",
    label: "Expires soon",
    color: "bg-orangeA-3 text-orangeA-11",
    icon: <CircleHalfDottedClock iconSize="sm-regular" className="text-orange-11" />,
    tooltip:
      "This key will expire in less than 24 hours. Rotate the key or extend its deadline to prevent disruptions.",
    priority: 2,
  },
  "validation-issues": {
    type: "validation-issues",
    label: "High Error Rate",
    color: "bg-warningA-3 text-warningA-11",
    icon: <ShieldKey iconSize="sm-regular" className="text-warning-11" />,
    tooltip:
      "This key is receiving many invalid requests (rate limited, unauthorized, etc.). Check logs to understand usage patterns.",
    priority: 3,
  },
  //TODO: Add a way to enable this through tooltip
  disabled: {
    type: "disabled",
    label: "Disabled",
    color: "bg-grayA-3 text-grayA-11",
    icon: <Ban iconSize="sm-regular" />,
    tooltip: "This key is currently disabled and cannot be used for verification.",
    priority: 0,
  },
  operational: {
    type: "operational",
    label: "Operational",
    color: "bg-successA-3 text-successA-11",
    icon: <CircleCheck iconSize="sm-regular" />,
    tooltip: "This key is operating normally.",
    priority: 99, // Lowest priority
  },
};
