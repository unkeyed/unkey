import type { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { Ban, CircleCheck, Lock, ShieldKey, TimeClock, TriangleWarning2 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { StatusBadge } from "./status-badge";

type LogOutcomeType = (typeof KEY_VERIFICATION_OUTCOMES)[number];
type LogOutcomeInfo = {
  type: LogOutcomeType;
  label: string;
  icon: React.ReactNode;
  tooltip: string;
};

const LOG_OUTCOME_DEFINITIONS: Record<LogOutcomeType, LogOutcomeInfo> = {
  VALID: {
    type: "VALID",
    label: "Valid",
    icon: <CircleCheck iconSize="sm-regular" />,
    tooltip: "The key was successfully verified.",
  },
  INSUFFICIENT_PERMISSIONS: {
    type: "INSUFFICIENT_PERMISSIONS",
    label: "Unauthorized",
    icon: <Lock iconSize="sm-regular" className="text-errorA-11" />,
    tooltip: "The key doesn't have sufficient permissions for this operation.",
  },
  RATE_LIMITED: {
    type: "RATE_LIMITED",
    label: "Ratelimited",
    icon: <TriangleWarning2 iconSize="sm-regular" className="text-warningA-11" />,
    tooltip: "The key has exceeded its rate limit.",
  },
  FORBIDDEN: {
    type: "FORBIDDEN",
    label: "Forbidden",
    icon: <Ban iconSize="sm-regular" className="text-errorA-11" />,
    tooltip: "The key is not authorized for this operation.",
  },
  DISABLED: {
    type: "DISABLED",
    label: "Disabled",
    icon: <ShieldKey iconSize="sm-regular" className="text-orangeA-11" />,
    tooltip: "The key has been disabled.",
  },
  EXPIRED: {
    type: "EXPIRED",
    label: "Expired",
    icon: <TimeClock iconSize="sm-regular" className="text-orangeA-11" />,
    tooltip: "The key has expired and is no longer valid.",
  },
  USAGE_EXCEEDED: {
    type: "USAGE_EXCEEDED",
    label: "Usage Exceeded",
    icon: <TriangleWarning2 iconSize="sm-regular" className="text-errorA-11" />,
    tooltip: "The key has exceeded its usage limit.",
  },
  "": {
    type: "",
    label: "Unknown",
    icon: <ShieldKey iconSize="sm-regular" />,
    tooltip: "Unknown verification status.",
  },
};

export type StatusStyle = {
  base: string;
  hover: string;
  selected: string;
  badge?: {
    default: string;
    selected: string;
  };
  focusRing: string;
};

export const STATUS_STYLES = {
  success: {
    base: "text-grayA-9",
    hover: "hover:text-grayA-11 hover:bg-grayA-3 dark:hover:text-grayA-12",
    selected: "text-grayA-12 bg-grayA-3",
    badge: {
      default: "bg-grayA-3 text-grayA-9 group-hover:bg-grayA-4",
      selected: "bg-grayA-4 text-grayA-11",
    },
    focusRing: "focus:ring-accent-7",
  },
  warning: {
    base: "text-warningA-11 bg-warning-2",
    hover: "hover:text-warningA-11 hover:bg-warningA-3",
    selected: "text-warningA-11 bg-warningA-3",
    badge: {
      default: "bg-warningA-3 text-warningA-11 group-hover:bg-warningA-4",
      selected: "bg-warningA-4 text-warningA-11",
    },
    focusRing: "focus:ring-warning-7",
  },
  blocked: {
    base: "text-orangeA-11 bg-orange-2",
    hover: "hover:text-orangeA-11 hover:bg-orangeA-3",
    selected: "text-orangeA-11 bg-orangeA-3",
    badge: {
      default: "bg-orangeA-3 text-orangeA-11 group-hover:bg-orangeA-4",
      selected: "bg-orangeA-4 text-orangeA-11",
    },
    focusRing: "focus:ring-orange-7",
  },
  error: {
    base: "text-errorA-11 bg-error-2",
    hover: "hover:text-errorA-11 hover:bg-errorA-3",
    selected: "text-errorA-11 bg-errorA-3",
    badge: {
      default: "bg-errorA-3 text-errorA-11 group-hover:bg-errorA-4",
      selected: "bg-errorA-4 text-errorA-11",
    },
    focusRing: "focus:ring-error-7",
  },
};

const getStatusType = (outcome: LogOutcomeType): keyof typeof STATUS_STYLES => {
  switch (outcome) {
    case "VALID":
      return "success";
    case "RATE_LIMITED":
      return "warning";
    case "DISABLED":
    case "EXPIRED":
      return "blocked";
    default:
      return "error";
  }
};

export const categorizeSeverity = (outcome: string): keyof typeof STATUS_STYLES => {
  switch (outcome) {
    case "VALID":
      return "success";
    case "RATE_LIMITED":
      return "warning";
    case "DISABLED":
    case "EXPIRED":
      return "blocked";
    case "INSUFFICIENT_PERMISSIONS":
    case "FORBIDDEN":
    case "USAGE_EXCEEDED":
      return "error";
    default:
      return "success";
  }
};

export const getStatusStyle = (log: KeyDetailsLog): StatusStyle => {
  const severity = categorizeSeverity(log.outcome);
  return STATUS_STYLES[severity];
};

type OutcomeProps = {
  log: KeyDetailsLog;
  selectedLog: KeyDetailsLog | null;
};

/**
 * Single status item for log like KEY_VERIFICATION_OUTCOMES
 * @param log for the row
 * @param selectedLog for log selected in table
 * @returns JSX element that shows status outcomes of log
 */

export const StatusCell = ({ log, selectedLog }: OutcomeProps) => {
  const isSelected = selectedLog?.request_id === log.request_id;
  const outcomeType =
    (log.outcome as LogOutcomeType) in LOG_OUTCOME_DEFINITIONS
      ? (log.outcome as LogOutcomeType)
      : "";
  const outcomeInfo = LOG_OUTCOME_DEFINITIONS[outcomeType];
  return (
    <InfoTooltip
      className="cursor-default"
      content={<p>{outcomeInfo.tooltip}</p>}
      position={{ side: "top", align: "center", sideOffset: 5 }}
    >
      <div className="flex gap-3 items-center">
        <StatusBadge
          primary={{
            label: outcomeInfo.label,
            color: isSelected
              ? (STATUS_STYLES[getStatusType(outcomeInfo.type)].badge?.selected ??
                "bg-grayA-3 text-grayA-9")
              : (STATUS_STYLES[getStatusType(outcomeInfo.type)].badge?.default ??
                "bg-grayA-3 text-grayA-9"),
            icon: outcomeInfo.icon,
          }}
        />
      </div>
    </InfoTooltip>
  );
};
