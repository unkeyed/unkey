import type { RuntimeLog } from "@/lib/schemas/runtime-logs.schema";

export type { RuntimeLog };

export type RuntimeLogsFilterField = "severity" | "message" | "startTime" | "endTime" | "since";

export type RuntimeLogsFilterOperator = "is";

export type RuntimeLogsFilter = {
  id: string;
  field: RuntimeLogsFilterField;
  operator: RuntimeLogsFilterOperator;
  value: string | number;
  metadata?: {
    colorClass?: string;
    label?: string;
  };
};
