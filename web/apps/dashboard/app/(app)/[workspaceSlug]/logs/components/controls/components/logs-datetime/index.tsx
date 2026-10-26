import { DEFAULT_OPTIONS } from "@/components/logs/datetime/constants";
import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { useWorkspaceLimits } from "@/hooks/use-workspace-limits";
import { getTimestampFromRelative } from "@/lib/duration";
import { DAY_IN_MS } from "@/lib/trpc/routers/utils/constants";
import { IconCalendarOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "cn";
import { useEffect, useMemo, useState } from "react";
import { useFilters } from "../../../../hooks/use-filters";

export const LogsDateTime = () => {
  const [title, setTitle] = useState<string | null>(null);
  const { filters, updateFilters } = useFilters();
  const { data: limits } = useWorkspaceLimits();
  const retentionDays = limits?.logsRetentionDaysMax.limit ?? null;

  // A preset longer than the log retention would chart mostly empty buckets
  const options = useMemo(() => {
    if (retentionDays === null) {
      return DEFAULT_OPTIONS;
    }
    const now = Date.now();
    return DEFAULT_OPTIONS.filter(
      (option) =>
        option.value === undefined ||
        now - getTimestampFromRelative(option.value, now) <= retentionDays * DAY_IN_MS,
    );
  }, [retentionDays]);

  useEffect(() => {
    if (!title) {
      setTitle("Last 12 hours");
    }
  }, [title]);

  const hasTimeFilters = filters.some((f) => ["startTime", "endTime", "since"].includes(f.field));
  const displayTitle = hasTimeFilters ? (title ?? "Loading...") : "Last 12 hours";

  const timeValues = filters
    .filter((f) => ["startTime", "endTime", "since"].includes(f.field))
    .reduce(
      (acc, f) => ({
        // biome-ignore lint/performance/noAccumulatingSpread: it's safe to spread
        ...acc,
        [f.field]: f.value,
      }),
      {},
    );

  return (
    <DatetimePopover
      // The popover keeps its preset list in state, so it remounts once the
      // retention is known
      key={retentionDays ?? "unlimited"}
      customOptions={options}
      minDate={
        retentionDays === null ? undefined : new Date(Date.now() - retentionDays * DAY_IN_MS)
      }
      maxDate={new Date()}
      initialTimeValues={timeValues}
      initialTitle={displayTitle}
      onDateTimeChange={(startTime, endTime, since) => {
        const activeFilters = filters.filter(
          (f) => !["endTime", "startTime", "since"].includes(f.field),
        );
        if (since !== undefined) {
          updateFilters([
            ...activeFilters,
            {
              field: "since",
              value: since,
              id: crypto.randomUUID(),
              operator: "is",
            },
          ]);
          return;
        }
        if (since === undefined && startTime) {
          activeFilters.push({
            field: "startTime",
            value: startTime,
            id: crypto.randomUUID(),
            operator: "is",
          });
          if (endTime) {
            activeFilters.push({
              field: "endTime",
              value: endTime,
              id: crypto.randomUUID(),
              operator: "is",
            });
          }
        }
        updateFilters(activeFilters);
      }}
      onSuggestionChange={setTitle}
    >
      <Button
        variant="ghost"
        size="md"
        className={cn(
          "data-popup-open:bg-gray-4 px-2 rounded-lg",
          displayTitle === "Loading..." ? "opacity-50" : "",
          displayTitle !== "Last 12 hours" ? "bg-gray-4" : "",
        )}
        aria-label="Filter logs by time"
        aria-haspopup="true"
        title="Press 'T' to toggle filters"
        disabled={displayTitle === "Loading..."}
      >
        <IconCalendarOutline18 className="text-gray-9 size-4" />
        <span className="text-gray-12 font-medium text-sm">{displayTitle}</span>
      </Button>
    </DatetimePopover>
  );
};
