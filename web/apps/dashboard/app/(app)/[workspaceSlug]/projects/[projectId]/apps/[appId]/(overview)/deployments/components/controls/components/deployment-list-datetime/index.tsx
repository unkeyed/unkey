import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { isTimeFilter, timeValuesOf, withTimeRange } from "@/components/logs/datetime/time-filters";
import { IconCalendarOutline18 } from "@unkey/icons";
import { useState } from "react";
import { useFilters } from "../../../../hooks/use-filters";
import { FilterTriggerButton } from "../filter-trigger-button";

const TITLE_EMPTY_DEFAULT = "Select Time Range";

export const DeploymentListDatetime = () => {
  const [title, setTitle] = useState<string | null>(TITLE_EMPTY_DEFAULT);
  const { filters, updateFilters } = useFilters();

  const timeFilters = filters.filter(isTimeFilter);
  const hasTimeFilters = timeFilters.length > 0;
  const isLoading = hasTimeFilters && title === null;
  const displayTitle = hasTimeFilters ? (title ?? "Loading...") : TITLE_EMPTY_DEFAULT;

  return (
    <DatetimePopover
      maxDate={new Date()}
      align="end"
      initialTimeValues={timeValuesOf(timeFilters)}
      onDateTimeChange={(startTime, endTime, since) =>
        updateFilters(withTimeRange(filters, { startTime, endTime, since }))
      }
      initialTitle={displayTitle}
      onSuggestionChange={setTitle}
    >
      <FilterTriggerButton
        aria-label="Filter deployments by time"
        aria-haspopup="true"
        disabled={isLoading}
        icon={<IconCalendarOutline18 className="size-4 text-gray-9 shrink-0" />}
        label={displayTitle}
        isActive={hasTimeFilters}
      />
    </DatetimePopover>
  );
};
