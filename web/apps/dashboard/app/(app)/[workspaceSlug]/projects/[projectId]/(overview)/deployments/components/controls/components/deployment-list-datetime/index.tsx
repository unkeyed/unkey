import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { Calendar } from "@unkey/icons";
import { useState } from "react";
import type { DeploymentListFilterValue } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";
import { FilterTriggerButton } from "../filter-trigger-button";

const TITLE_EMPTY_DEFAULT = "Select Time Range";
const TIME_FIELDS: readonly string[] = ["startTime", "endTime", "since"];

export const DeploymentListDatetime = () => {
  const [title, setTitle] = useState<string | null>(TITLE_EMPTY_DEFAULT);
  const { filters, updateFilters } = useFilters();

  const hasTimeFilters = filters.some((f) => TIME_FIELDS.includes(f.field));
  const isLoading = hasTimeFilters && title === null;
  const displayTitle = hasTimeFilters ? (title ?? "Loading...") : TITLE_EMPTY_DEFAULT;

  const timeValues = Object.fromEntries(
    filters.filter((f) => TIME_FIELDS.includes(f.field)).map((f) => [f.field, f.value]),
  );

  return (
    <DatetimePopover
      maxDate={new Date()}
      align="end"
      initialTimeValues={timeValues}
      onDateTimeChange={(startTime, endTime, since) => {
        const next: DeploymentListFilterValue[] = filters.filter(
          (f) => !TIME_FIELDS.includes(f.field),
        );
        if (since !== undefined) {
          next.push({ field: "since", value: since, id: crypto.randomUUID(), operator: "is" });
        } else if (startTime) {
          next.push({
            field: "startTime",
            value: startTime,
            id: crypto.randomUUID(),
            operator: "is",
          });
          if (endTime) {
            next.push({
              field: "endTime",
              value: endTime,
              id: crypto.randomUUID(),
              operator: "is",
            });
          }
        }
        updateFilters(next);
      }}
      initialTitle={displayTitle}
      onSuggestionChange={setTitle}
    >
      <FilterTriggerButton
        aria-label="Filter deployments by time"
        aria-haspopup="true"
        disabled={isLoading}
        icon={<Calendar iconSize="md-medium" className="text-gray-9 shrink-0" />}
        label={displayTitle}
        isActive={hasTimeFilters}
      />
    </DatetimePopover>
  );
};
