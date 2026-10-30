import { HISTORY_INSTANCES_MAX } from "@unkey/clickhouse/src/app-metrics/resources";

export function HistoryNotice() {
  return (
    <p className="text-xs text-gray-11">
      Showing live instances and the {HISTORY_INSTANCES_MAX} most recent stopped instances. Resource
      totals include all instances.
    </p>
  );
}
