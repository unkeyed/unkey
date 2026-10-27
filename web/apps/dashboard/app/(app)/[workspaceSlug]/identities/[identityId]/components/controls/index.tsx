"use client";
import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { HOUR_PLUS_OPTIONS } from "@/components/logs/datetime/constants";
import { TimeRangeFilter } from "@/components/time-range-filter";
import { useFilters } from "../../hooks/use-filters";
import { LogsFilters } from "./components/logs-filters";
import { LogsLiveSwitch } from "./components/logs-live-switch";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function IdentityDetailsLogsControls({
  identityId,
}: {
  identityId: string;
}) {
  const { filters, updateFilters } = useFilters();
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch identityId={identityId} />
        <LogsFilters />
        <TimeRangeFilter
          filters={filters}
          updateFilters={updateFilters}
          look="logsCompact"
          customOptions={HOUR_PLUS_OPTIONS}
        />
      </ControlsLeft>
      <ControlsRight>
        <LogsLiveSwitch />
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
