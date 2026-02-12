"use client";

import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { RuntimeLogsDateTime } from "./components/runtime-logs-datetime";
import { RuntimeLogsFilters } from "./components/runtime-logs-filters";
import { RuntimeLogsRefresh } from "./components/runtime-logs-refresh";
import { RuntimeLogsSearch } from "./components/runtime-logs-search";

export function RuntimeLogsControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <RuntimeLogsSearch />
        <RuntimeLogsFilters />
        <RuntimeLogsDateTime />
      </ControlsLeft>
      <ControlsRight>
        <RuntimeLogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
