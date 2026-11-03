import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { STATUS_STYLES, type StatusStyle } from "@unkey/ui";
import { cn } from "cn";

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
    base: "text-gray-12",
    hover: "hover:text-gray-12 hover:bg-success-3",
    selected: "bg-success-3 text-success-11",
    badge: {
      default: "bg-success-4 text-success-11 group-hover:bg-success-5",
      selected: "bg-success-5 text-success-12 hover:bg-success-5",
    },
    focusRing: "focus:ring-success-7",
  },
  update: {
    base: "text-gray-12",
    hover: "hover:text-gray-12 hover:bg-warning-3",
    selected: "bg-warning-3 text-warning-11",
    badge: {
      default: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
      selected: "bg-warning-5 text-warning-12 hover:bg-warning-5",
    },
    focusRing: "focus:ring-warning-7",
  },
  delete: {
    base: "text-gray-12",
    hover: "hover:text-gray-12 hover:bg-orange-3",
    selected: "bg-orange-3 text-orange-11",
    badge: {
      default: "bg-orange-4 text-orange-11 group-hover:bg-orange-5",
      selected: "bg-orange-5 text-orange-12 hover:bg-orange-5",
    },
    focusRing: "focus:ring-orange-7",
  },
  other: STATUS_STYLES,
} satisfies Record<ReturnType<typeof getEventType>, StatusStyle>;

export const getAuditStatusStyle = (item: AuditLog) => {
  const eventType = getEventType(item.auditLog.event);
  return AUDIT_STATUS_STYLES[eventType];
};

export const getAuditRowClassName = (item: AuditLog, selectedLog: AuditLog | null) => {
  const eventType = getEventType(item.auditLog.event);
  const style = AUDIT_STATUS_STYLES[eventType];
  const isSelected = item.auditLog.id === selectedLog?.auditLog.id;
  const logSelected = Boolean(selectedLog);

  return cn(
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-hidden focus:ring-1",
    style.focusRing,
    isSelected && style.selected,
    logSelected && {
      "opacity-50 z-0": !isSelected,
      "opacity-100 z-10": isSelected,
    },
  );
};

export const getAuditSelectedClassName = (item: AuditLog, isSelected: boolean) => {
  if (!isSelected) {
    return "";
  }

  const style = AUDIT_STATUS_STYLES[getEventType(item.auditLog.event)];
  return style.selected;
};
