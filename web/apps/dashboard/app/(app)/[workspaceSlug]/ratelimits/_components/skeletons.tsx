import { Clock, ProgressBar } from "@unkey/icons";

export const ChartSkeleton = () => {
  return <div className="h-[140px] bg-grayA-2 p-4 flex items-end gap-1" />;
};

export const MetricStatsSkeleton = () => {
  return (
    <>
      <div className="flex gap-[14px] items-center">
        <div className="flex flex-col gap-1">
          <div className="flex gap-2 items-center h-4">
            <div className="bg-accent-8 rounded h-[10px] w-1 animate-pulse" />
            <div className="h-3 w-16 bg-grayA-3 rounded animate-pulse" />
          </div>
        </div>
        <div className="flex flex-col gap-1">
          <div className="flex gap-2 items-center h-4">
            <div className="bg-orange-9 rounded h-[10px] w-1 animate-pulse" />
            <div className="h-3 w-16 bg-grayA-3 rounded animate-pulse" />
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2 min-w-0 max-w-[40%] h-4">
        <Clock className="text-accent-11 flex-shrink-0 opacity-30" />
        <div className="h-3 w-10 bg-grayA-3 rounded animate-pulse" />
      </div>
    </>
  );
};

export const NamespaceCardSkeleton = () => {
  return (
    <div className="flex flex-col border border-gray-6 rounded-xl overflow-hidden">
      <ChartSkeleton />
      <div className="p-4 md:p-6 border-t border-gray-6 flex flex-col gap-2">
        <div className="flex justify-between items-center">
          <div className="flex flex-col flex-grow min-w-0">
            <div className="flex gap-2 md:gap-3 items-center h-6">
              <div className="flex-shrink-0 opacity-30">
                <ProgressBar className="text-accent-11" />
              </div>
              <div className="h-5 w-32 bg-grayA-3 rounded animate-pulse" />
            </div>
          </div>
        </div>
        <div className="flex items-center w-full justify-between gap-3 md:gap-4 mt-2">
          <MetricStatsSkeleton />
        </div>
      </div>
    </div>
  );
};
