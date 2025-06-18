"use client";

export const SettingsClientSkeleton = () => {
  return (
    <div className="py-3 w-full flex items-center justify-center">
      <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
        {/* Header skeleton */}
        <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
          API Settings
        </div>
        <div className="flex flex-col w-full gap-6">
          {/* Name and ID cards skeleton */}
          <div>
            {/* Name card skeleton - border="top" */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-t-xl border-b-1">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="h-5 w-16 bg-grayA-3 rounded animate-pulse transition-all" />
                <div className="h-3 w-72 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
                <div className="h-3 w-72 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
                <div className="h-3 w-72 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
              </div>
              <div className="flex w-full lg:w-[420px] h-full justify-end items-center">
                <div className="flex flex-row justify-end items-center w-full gap-x-2">
                  <div className="h-9 w-64 bg-grayA-3 animate-pulse transition-all border border-gray-5 rounded-lg" />
                  <div className="h-9 w-[62px] bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </div>
              </div>
            </div>
            {/* API ID card skeleton - border="bottom" */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-b-xl border-t-0">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="h-5 w-12 bg-grayA-3 rounded animate-pulse transition-all" />
                <div className="h-3 w-72 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
                <div className="h-3 w-72 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
              </div>
              <div className="flex w-full lg:w-[420px] h-full justify-end items-center">
                <div className="flex flex-row justify-end items-center gap-x-2">
                  <div className="h-9 w-[322px] bg-grayA-3 animate-pulse transition-all border border-gray-5 rounded-lg" />
                </div>
              </div>
            </div>
          </div>

          {/* Default bytes and prefix cards skeleton */}
          <div>
            {/* Default bytes card skeleton - border="top" */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-t-xl border-b-1">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="h-5 w-24 bg-grayA-3 rounded animate-pulse transition-all" />
                <div className="h-4 w-80 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
              </div>
              <div className="flex w-full lg:w-[420px] h-full justify-end items-center">
                <div className="flex flex-row justify-end items-center w-full gap-x-2">
                  <div className="h-9 w-64 bg-grayA-3 animate-pulse transition-all border border-gray-5 rounded-lg" />
                  <div className="h-9 w-[62px] bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </div>
              </div>
            </div>
            {/* Default prefix card skeleton - border="bottom" */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-b-xl border-t-0">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="h-5 w-24 bg-grayA-3 rounded animate-pulse transition-all" />
                <div className="h-4 w-80 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
              </div>
              <div className="flex w-full lg:w-[420px] h-full justify-end items-center">
                <div className="flex flex-row justify-end items-center w-full gap-x-2">
                  <div className="h-9 w-64 bg-grayA-3 animate-pulse transition-all border border-gray-5 rounded-lg" />
                  <div className="h-9 w-[62px] bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </div>
              </div>
            </div>
          </div>

          {/* IP Whitelist card skeleton - border="both" */}
          <div className="w-full">
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-xl">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="flex items-center justify-start gap-2.5 h-6">
                  <div className="h-5 w-20 bg-grayA-3 rounded animate-pulse transition-all" />
                  <div className="h-4 w-12 bg-grayA-3 rounded animate-pulse transition-all" />
                </div>
                <div className="h-3 w-72 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
                <div className="h-3 w-72 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
                <div className="h-3 w-72 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
              </div>
              <div className="flex w-full h-full justify-end items-start">
                <div className="flex flex-row justify-end items-start w-full gap-x-2">
                  <div className="h-9 w-48 bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </div>
              </div>
            </div>
          </div>

          {/* Delete protection and delete API cards skeleton */}
          <div>
            {/* Delete protection card skeleton - border="top" */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-t-xl border-b-1">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="inline-flex gap-2">
                  <div className="h-5 w-32 bg-grayA-3 rounded animate-pulse transition-all" />
                  <div className="h-4 w-16 bg-grayA-3 rounded animate-pulse transition-all" />
                </div>
                <div className="h-4 w-80 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
              </div>
              <div className="flex w-full lg:items-center justify-end">
                <div className="h-9 w-48 bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
              </div>
            </div>
            {/* Delete API card skeleton - border="bottom" */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-b-xl border-t-0">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="inline-flex gap-2">
                  <div className="h-5 w-20 bg-grayA-3 rounded animate-pulse transition-all" />
                  <div className="h-4 w-12 bg-grayA-3 rounded animate-pulse transition-all" />
                </div>
                <div className="h-4 w-96 bg-grayA-3 rounded animate-pulse transition-all mt-1" />
                <div className="h-4 w-72 bg-grayA-3 rounded animate-pulse transition-all" />
              </div>
              <div className="flex w-full justify-end">
                <div className="h-9 w-[104px] bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
