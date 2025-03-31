import { LogsDateTime } from "./components/logs-datetime";
import { LogsFilters } from "./components/logs-filters";
import { LogsLiveSwitch } from "./components/logs-live-switch";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function RatelimitLogsControls() {
  return (
    <div className="flex flex-col border-b border-gray-4 ">
      <div className="px-3 py-1 w-full justify-between flex items-center min-h-10">
        <div className="flex gap-2">
          <div className="flex gap-2 items-center">
            <LogsSearch />
          </div>
          <div className="flex gap-2 items-center">
            <LogsFilters />
          </div>
          <div className="flex gap-2 items-center">
            <LogsDateTime />
          </div>
        </div>

        <div className="flex gap-2">
          <div className="flex gap-2 items-center">
            <LogsLiveSwitch />
          </div>
          <div className="flex gap-2 items-center">
            <LogsRefresh />
          </div>
        </div>
      </div>
    </div>
  );
}
