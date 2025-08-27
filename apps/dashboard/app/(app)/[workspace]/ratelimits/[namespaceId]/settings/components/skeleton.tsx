export const SettingsClientSkeleton = () => {
  return (
    <div className="py-3 w-full flex items-center justify-center">
      <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
        {/* Header skeleton */}
        <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
          Ratelimit Settings
        </div>

        <div className="flex flex-col w-full gap-6">
          {/* Namespace name card skeleton */}
          <div>
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-t-xl border-b-1">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="h-5 w-32 bg-grayA-3 rounded animate-pulse transition-all" />
                <div className="h-4 w-64 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
                <div className="h-4 w-44 bg-grayA-3 rounded animate-pulse transition-all" />
              </div>
              <div className="flex w-full lg:w-[420px] h-full justify-end items-end">
                <div className="flex flex-row justify-end items-center gap-x-2 mt-2 h-9">
                  <div className="h-full w-64 bg-grayA-3 animate-pulse transition-all border border-gray-5 rounded-lg" />
                  <div className="h-full w-[62px] bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </div>
              </div>
            </div>

            {/* Namespace ID card skeleton */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-b-xl border-t-0">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="h-5 w-28 bg-grayA-3 rounded animate-pulse transition-all" />
                <div className="h-4 w-56 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
              </div>
              <div className="flex w-full lg:w-[320px] h-full justify-end items-end">
                <div className="flex flex-row justify-end items-center gap-x-2 mt-1">
                  <div className="h-9 w-[320px] bg-grayA-3 animate-pulse transition-all border border-gray-5 rounded-lg" />
                </div>
              </div>
            </div>
          </div>

          {/* Delete namespace card skeleton */}
          <div className="w-full">
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-xl">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="h-5 w-28 bg-grayA-3 rounded animate-pulse transition-all" />
                <div className="h-4 w-52 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
                <div className="h-4 w-48 bg-grayA-3 rounded animate-pulse transition-all" />
              </div>
              <div className="flex w-full lg:w-[320px] h-full justify-end items-end">
                <div className="w-full flex justify-end lg:mt-3">
                  <div className="h-9 w-[156px] bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
