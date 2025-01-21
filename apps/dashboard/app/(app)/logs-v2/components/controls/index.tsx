import { Calendar, CircleCarretRight, Refresh3 } from "@unkey/icons";
import { LogsDisplay } from "./components/logs-display";
import { LogsFilters } from "./components/logs-filters";
import { LogsSearch } from "./components/logs-search";

export function LogsControls() {
  return (
    <div className="flex flex-col border-b border-gray-4 ">
      <div className="px-3 py-2 w-full justify-between flex items-center min-h-10">
        <div className="flex gap-2">
          <div className="flex gap-2 items-center">
            <LogsSearch />
          </div>
          <div className="flex gap-2 items-center">
            <LogsFilters />
          </div>
          <div className="flex gap-2 items-center px-2">
            <Calendar className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-[13px]">Last 24 hours</span>
          </div>
        </div>

        <div className="flex gap-2">
          <div className="flex gap-2 items-center px-2">
            <CircleCarretRight className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-[13px]">Live</span>
          </div>
          <div className="flex gap-2 items-center px-2">
            <Refresh3 className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-[13px]">Refresh</span>
          </div>
          <div className="flex gap-2 items-center">
            <LogsDisplay />
          </div>
        </div>
      </div>
    </div>
  );
}
