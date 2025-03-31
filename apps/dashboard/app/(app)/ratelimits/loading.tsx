import { Skeleton } from "@/components/ui/skeleton";
import { Clock, Gauge, Magnifier, ProgressBar, Refresh3 } from "@unkey/icons";

export default function Loading() {
  return (
    <div>
      {/* Navigation Bar */}
      <nav className="w-full p-4 border-b border-gray-4 bg-background justify-between flex">
        <nav aria-label="breadcrumb" className="flex">
          <ol className="flex items-center gap-3">
            <li className="mr-1">
              {/* biome-ignore lint/a11y/useButtonType: <explanation> */}
              <button className="inline-flex group relative duration-150 items-center justify-center gap-2 whitespace-nowrap text-sm transition-colors focus-visible:ring-1 focus-visible:ring-ring [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 disabled:cursor-not-allowed text-gray-12 border border-grayA-6 rounded-md focus:border-grayA-12 focus:ring-4 focus:ring-gray-6 focus-visible:outline-none focus:ring-offset-0 drop-shadow-button disabled:border disabled:border-solid disabled:border-grayA-5 disabled:text-grayA-7 active:bg-grayA-3 size-6 p-0 [&>svg]:size-[18px] bg-gray-4 hover:bg-gray-5">
                <div className="w-full h-full flex items-center justify-center gap-2 transition-opacity duration-200 opacity-100">
                  <Gauge />
                </div>
              </button>
            </li>
            <li className="flex items-center gap-3">
              <a
                href="/ratelimits"
                className="text-sm transition-colors text-accent-10 hover:text-accent-11"
                aria-current="page"
              >
                Ratelimits
              </a>
            </li>
          </ol>
        </nav>
        <div className="flex items-center gap-3">
          <div className="flex flex-col gap-1">
            {/* biome-ignore lint/a11y/useButtonType: <explanation> */}
            <button
              className="inline-flex group relative duration-150 items-center justify-center gap-2 whitespace-nowrap text-sm transition-colors focus-visible:ring-1 focus-visible:ring-ring [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 disabled:cursor-not-allowed p-2 text-gray-12 border border-grayA-6 rounded-md focus:border-grayA-12 focus:ring-4 focus:ring-gray-6 focus-visible:outline-none focus:ring-offset-0 drop-shadow-button disabled:border disabled:border-solid disabled:border-grayA-5 disabled:text-grayA-7 active:bg-grayA-3 h-8 bg-grayA-2 hover:bg-grayA-3"
              title="Create New Root Key"
            >
              <div className="w-full h-full flex items-center justify-center gap-2 transition-opacity duration-200 opacity-100">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="18"
                  height="18"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="lucide lucide-plus w-4 h-4"
                >
                  <path d="M5 12h14" />
                  <path d="M12 5v14" />
                </svg>
                Create new namespace
              </div>
            </button>
          </div>
        </div>
      </nav>
      <div className="flex flex-col">
        {/* Search and Controls Row - Using LogsLLMSearch and RefreshButton structures */}
        <div className="flex flex-col border-b border-gray-4">
          <div className="px-3 py-1 w-full justify-between flex items-center">
            <div className="flex gap-2">
              <div className="flex gap-2 items-center">
                {/* LogsLLMSearch - Based on the component structure */}
                <div className="group relative" data-testid="logs-llm-search">
                  <div className="px-2 flex items-center w-80 gap-2 border rounded-lg py-1 h-8 border-none hover:bg-gray-3 transition-all duration-200">
                    <div className="flex items-center gap-2 w-80">
                      <div className="flex-shrink-0">
                        <Magnifier />
                      </div>
                      <div className="flex-1">
                        <span className="text-accent-9 text-[13px] font-medium">
                          Search namespaces
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div className="flex gap-2 items-center">
                {/* Date filter */}
                <div className="flex flex-row items-center">
                  <div className="group">
                    {/* biome-ignore lint/a11y/useButtonType: <explanation> */}
                    <button className="inline-flex group relative duration-150 items-center justify-center gap-2 whitespace-nowrap text-sm transition-colors p-2 text-gray-12 bg-transparent h-8 px-2 rounded-lg">
                      <div className="w-full h-full flex items-center justify-center gap-2">
                        <Clock />
                        <span className="text-gray-12 font-medium text-[13px]">Last 12 hours</span>
                      </div>
                    </button>
                  </div>
                </div>
              </div>
            </div>
            <div className="flex gap-2">
              {/* RefreshButton - Based on the component structure but disabled */}
              <div>
                {/* biome-ignore lint/a11y/useButtonType: <explanation> */}
                <button
                  disabled
                  className="group relative duration-150 gap-2 whitespace-nowrap text-sm transition-colors focus-visible:ring-1 focus-visible:ring-ring [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 disabled:cursor-not-allowed p-2 bg-transparent focus:border-grayA-12 focus:ring-4 focus:ring-gray-6 focus-visible:outline-none focus:ring-offset-0 drop-shadow-button disabled:border disabled:border-grayA-4 disabled:text-grayA-7 h-8 flex w-full items-center justify-center rounded-lg border border-gray-4"
                >
                  <div className="w-full h-full flex items-center justify-center gap-2 transition-opacity duration-200 opacity-100">
                    <Refresh3 className="text-grayA-7" />
                    <span className="font-medium text-grayA-7 text-[13px] relative z-10">
                      Refresh
                    </span>
                    <div className="inline-flex group relative bg-gray-3 text-gray-9 border-gray-8 border text-xs h-5 px-1.5 min-w-[24px] rounded">
                      <div className="flex items-center justify-center">
                        <div>
                          CTRL+<span className="font-mono">R</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Cards Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5 w-full p-5">
          {Array.from({ length: 8 }).map((_, index) => (
            <div
              // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
              key={index}
              className="flex flex-col border border-gray-6 rounded-lg overflow-hidden"
            >
              {/* Chart area - 140px height */}
              <div className="h-[140px] bg-grayA-2" />
              {/* Bottom section with link, name, and stats */}
              <div className="p-4 md:p-6 border-t border-gray-6 flex flex-col gap-2">
                {/* Name and icon row */}
                <div className="flex justify-between items-center">
                  <div className="flex flex-col flex-grow min-w-0">
                    <div className="flex gap-2 md:gap-3 items-center">
                      <span className="flex-shrink-0">
                        <ProgressBar className="text-accent-11 w-5 h-5" />
                      </span>
                      <Skeleton className="w-48 h-5 rounded-lg" />
                    </div>
                  </div>
                </div>
                {/* Stats row */}
                <div className="flex items-center w-full justify-between gap-3 md:gap-4 mt-2">
                  {/* Left side - metrics inline */}
                  <div className="flex gap-[14px] items-center">
                    <div className="flex gap-2 items-center">
                      <div className="bg-accent-8 rounded h-[10px] w-1" />
                      <div className="text-accent-12 text-xs font-medium">0</div>
                      <div className="text-accent-9 text-[11px] leading-4">PASSED</div>
                    </div>
                    <div className="flex gap-2 items-center">
                      <div className="bg-orange-9 rounded h-[10px] w-1" />
                      <div className="text-accent-12 text-xs font-medium">0</div>
                      <div className="text-accent-9 text-[11px] leading-4">BLOCKED</div>
                    </div>
                  </div>
                  {/* Right side - clock */}
                  <div className="flex items-center gap-2 min-w-0 max-w-[40%]">
                    <Clock className="text-accent-11 flex-shrink-0 w-4 h-4" />
                    <span className="text-xs text-accent-9 truncate">No data</span>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
