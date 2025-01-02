import {
  Magnifier,
  BarsFilter,
  CircleCarretRight,
  Refresh3,
  Calendar,
  Sliders,
} from "@unkey/icons";

export function LogsFilters() {
  return (
    <div className="flex flex-col border-b border-gray-4">
      <div className="px-3 py-2 w-full justify-between flex items-center">
        <div className="flex gap-2">
          <div className="flex gap-2 items-center px-2">
            <Magnifier className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-xs">
              Search logs...
            </span>
          </div>
          <div className="flex gap-2 items-center px-2">
            <BarsFilter className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-xs">Filter</span>
          </div>
          <div className="flex gap-2 items-center px-2">
            <Calendar className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-xs">
              Last 24 hours
            </span>
          </div>
        </div>

        <div className="flex gap-2">
          <div className="flex gap-2 items-center px-2">
            <CircleCarretRight className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-xs">Live</span>
          </div>
          <div className="flex gap-2 items-center px-2">
            <Refresh3 className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-xs">Refresh</span>
          </div>
          <div className="flex gap-2 items-center px-2">
            <Sliders className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-xs">Display</span>
          </div>
        </div>
      </div>
    </div>
  );
}
