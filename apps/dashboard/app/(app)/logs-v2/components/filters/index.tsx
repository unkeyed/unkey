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
    <div className="flex flex-col border-b border-gray-4 ">
      <div className="px-3 py-2 w-full justify-between flex items-center min-h-10">
        <div className="flex gap-2">
          <div className="flex gap-2 items-center px-2">
            <Magnifier className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-[13px]">
              Search logs...
            </span>
          </div>
          <div className="flex gap-2 items-center px-2">
            <BarsFilter className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-[13px]">
              Filter
            </span>
          </div>
          <div className="flex gap-2 items-center px-2">
            <Calendar className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-[13px]">
              Last 24 hours
            </span>
          </div>
        </div>

        <div className="flex gap-2">
          <div className="flex gap-2 items-center px-2">
            <CircleCarretRight className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-[13px]">Live</span>
          </div>
          <div className="flex gap-2 items-center px-2">
            <Refresh3 className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-[13px]">
              Refresh
            </span>
          </div>
          <div className="flex gap-2 items-center px-2">
            <Sliders className="text-accent-9 size-4" />
            <span className="text-accent-12 font-medium text-[13px]">
              Display
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
