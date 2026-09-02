import type { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import {
  IconBanOutline12,
  IconCircleCheckOutline12,
  IconClockOutline12,
  IconLockOutline12,
  IconTriangleWarningOutline12,
} from "nucleo-ui-outline-12";
import { IconShieldKeyOutline18 } from "nucleo-ui-outline-18";

export type LogOutcomeType = (typeof KEY_VERIFICATION_OUTCOMES)[number];

export type LogOutcomeInfo = {
  type: LogOutcomeType;
  label: string;
  icon: React.ReactNode;
  tooltip: string;
};

export const LOG_OUTCOME_DEFINITIONS: Record<LogOutcomeType, LogOutcomeInfo> = {
  VALID: {
    type: "VALID",
    label: "Valid",
    icon: <IconCircleCheckOutline12 />,
    tooltip: "The key was successfully verified.",
  },
  INSUFFICIENT_PERMISSIONS: {
    type: "INSUFFICIENT_PERMISSIONS",
    label: "Unauthorized",
    icon: <IconLockOutline12 className="text-errorA-11" />,
    tooltip: "The key doesn't have sufficient permissions for this operation.",
  },
  RATE_LIMITED: {
    type: "RATE_LIMITED",
    label: "Ratelimited",
    icon: <IconTriangleWarningOutline12 className="text-warningA-11" />,
    tooltip: "The key has exceeded its rate limit.",
  },
  FORBIDDEN: {
    type: "FORBIDDEN",
    label: "Forbidden",
    icon: <IconBanOutline12 className="text-errorA-11" />,
    tooltip: "The key is not authorized for this operation.",
  },
  DISABLED: {
    type: "DISABLED",
    label: "Disabled",
    icon: <IconShieldKeyOutline18 className="size-3 text-orangeA-11" />,
    tooltip: "The key has been disabled.",
  },
  EXPIRED: {
    type: "EXPIRED",
    label: "Expired",
    icon: <IconClockOutline12 className="text-orangeA-11" />,
    tooltip: "The key has expired and is no longer valid.",
  },
  USAGE_EXCEEDED: {
    type: "USAGE_EXCEEDED",
    label: "Usage Exceeded",
    icon: <IconTriangleWarningOutline12 className="text-errorA-11" />,
    tooltip: "The key has exceeded its usage limit.",
  },
  "": {
    type: "",
    label: "Unknown",
    icon: <IconShieldKeyOutline18 className="size-3" />,
    tooltip: "Unknown verification status.",
  },
};

export const getStatusType = (
  outcome: LogOutcomeType,
): "success" | "warning" | "blocked" | "error" => {
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
