import type { RequestLogsFilterOperator } from "@/lib/schemas/request-logs.filter.schema";
import { RequestLogsTextFilter } from "./request-logs-text-filter";

const OPTIONS = ["is", "startsWith", "contains"] as const;

function validatePath(operator: RequestLogsFilterOperator, value: string): string | null {
  if ((operator === "contains" || operator === "startsWith") && value.length < 3) {
    return "Prefix and contains need at least 3 characters. Use is for shorter paths.";
  }
  if (value.length > 2_048) {
    return "Path must be 2,048 characters or fewer.";
  }
  return null;
}

export const RequestPathsFilter = () => {
  return (
    <RequestLogsTextFilter field="paths" label="Path" operators={OPTIONS} validate={validatePath} />
  );
};
