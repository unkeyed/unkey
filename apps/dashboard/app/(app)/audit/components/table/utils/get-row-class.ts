import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { cn } from "@/lib/utils";

/**
 * Determines the event type based on the event name
 */
export const getEventType = (event: string): "create" | "update" | "delete" | "other" => {
  const eventLower = event.toLowerCase();

  if (eventLower.includes("create") || eventLower.includes("add")) {
    return "create";
  }

  if (
    eventLower.includes("update") ||
    eventLower.includes("edit") ||
    eventLower.includes("modify")
  ) {
    return "update";
  }

  if (eventLower.includes("delete") || eventLower.includes("remove")) {
    return "delete";
  }

  return "other";
};

export const AUDIT_STATUS_STYLES = {
  create: {
    base: "text-grayA-9",
    hover: "hover:bg-success-3",
    selected: "bg-success-3 text-success-11",
    badge: {
      default: "bg-success-4 text-success-11 group-hover:bg-success-5",
      selected: "bg-success-5 text-success-12 hover:bg-success-5",
    },
    focusRing: "focus:ring-success-7",
  },
  update: {
    base: "text-grayA-9",
    hover: "hover:bg-warning-3",
    selected: "bg-warning-3 text-warning-11",
    badge: {
      default: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
      selected: "bg-warning-5 text-warning-12 hover:bg-warning-5",
    },
    focusRing: "focus:ring-warning-7",
  },
  delete: {
    base: "text-grayA-9",
    hover: "hover:bg-orange-3",
    selected: "bg-orange-3 text-orange-11",
    badge: {
      default: "bg-orange-4 text-orange-11 group-hover:bg-orange-5",
      selected: "bg-orange-5 text-orange-12 hover:bg-orange-5",
    },
    focusRing: "focus:ring-orange-7",
  },
  other: {
    base: "text-grayA-9",
    hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-3",
    selected: "text-accent-12 bg-grayA-3 hover:text-accent-12",
    badge: {
      default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5",
      selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5",
    },
    focusRing: "focus:ring-accent-7",
  },
};

/**
 * Get the style configuration for an audit log entry
 */
export const getAuditStatusStyle = (item: AuditLog) => {
  const eventType = getEventType(item.auditLog.event);
  return AUDIT_STATUS_STYLES[eventType];
};

/**
 * Get the row class name for an audit log entry
 */
export const getAuditRowClassName = (item: AuditLog, isSelected: boolean, logSelected: boolean) => {
  const eventType = getEventType(item.auditLog.event);
  const style = AUDIT_STATUS_STYLES[eventType];

  return cn(
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
    logSelected && {
      "opacity-50 z-0": !isSelected,
      "opacity-100 z-10": isSelected,
    },
  );
};

/**
 * Get the selected class name for an audit log entry
 */
export const getAuditSelectedClassName = (item: AuditLog, isSelected: boolean) => {
  if (!isSelected) {
    return "";
  }

  const style = AUDIT_STATUS_STYLES[getEventType(item.auditLog.event)];
  return style.selected;
};
