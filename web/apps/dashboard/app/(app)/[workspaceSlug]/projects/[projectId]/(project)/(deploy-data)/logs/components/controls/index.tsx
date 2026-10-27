"use client";

import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { TimeRangeFilter } from "@/components/time-range-filter";
import { useRuntimeLogsFilters } from "../../hooks/use-runtime-logs-filters";
import { RuntimeLogsFilters } from "./components/runtime-logs-filters";
import { RuntimeLogsLiveSwitch } from "./components/runtime-logs-live-switch";
import { RuntimeLogsRefresh } from "./components/runtime-logs-refresh";
import { RuntimeLogsSearch } from "./components/runtime-logs-search";

export function RuntimeLogsControls() {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  return (
    <ControlsContainer>
      <ControlsLeft>
        <RuntimeLogsSearch />
        <RuntimeLogsFilters />
        <TimeRangeFilter
          filters={filters}
          updateFilters={updateFilters}
          look="logs"
          defaultTitle="Last 6 hours"
        />
      </ControlsLeft>
      <ControlsRight>
        <RuntimeLogsLiveSwitch />
        <RuntimeLogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
