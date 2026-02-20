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
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-t-xl border-b">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="font-medium text-accent-12 leading-5 tracking-normal">
                  <div className="h-5 w-16 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                </div>
                <div className="font-normal text-accent-11 text-[13px] leading-5 tracking-normal">
                  <div className="h-10 w-80 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                </div>
              </div>
              <div className="flex w-[420px]">
                <form className="flex flex-row justify-end items-center gap-x-2 h-9 w-full">
                  <div className="h-9 min-w-64 items-end bg-grayA-3 animate-pulse transition-all border border-grayA-5 rounded-lg" />
                  <div className="h-9 w-[62px] bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </form>
              </div>
            </div>
            {/* API ID card skeleton - no border change, just content */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-b-xl border-t-0">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="font-medium text-accent-12 leading-5 tracking-normal">
                  <div className="h-5 w-12 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                </div>
                <div className="font-normal text-accent-11 text-[13px] leading-5 tracking-normal">
                  <div className="h-5 w-72 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                </div>
              </div>
              <div className="flex w-[322px]">
                <div className="flex flex-row justify-end items-center gap-x-2 h-9 w-full">
                  <div className="h-9 w-[380px] bg-grayA-3 animate-pulse transition-all border border-grayA-5 rounded-lg" />
                </div>
              </div>
            </div>
          </div>

          {/* Default bytes and prefix cards skeleton */}
          <div>
            {/* Default bytes card skeleton - border="top" */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-t-xl border-b">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="font-medium text-accent-12 leading-5 tracking-normal">
                  <div className="h-5 w-24 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                </div>
                <div className="font-normal text-accent-11 text-[13px] leading-5 tracking-normal">
                  <div className="max-w-[380px]">
                    <div className="h-10 w-80 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                  </div>
                </div>
              </div>
              <div className="flex w-[420px] h-full justify-end items-end">
                <form className="flex flex-row justify-end items-center gap-x-2 h-9 w-full">
                  <input type="hidden" />
                  <div className="h-9 min-w-64 items-end bg-grayA-3 animate-pulse transition-all border border-grayA-5 rounded-lg" />
                  <div className="size-lg variant-primary px-3.5 h-9 w-[62px] bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </form>
              </div>
            </div>
            {/* Default prefix card skeleton - border="bottom" */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-b-xl border-t-0">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="font-medium text-accent-12 leading-5 tracking-normal">
                  <div className="h-5 w-24 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                </div>
                <div className="font-normal text-accent-11 text-[13px] leading-5 tracking-normal">
                  <div className="max-w-[380px]">
                    <div className="h-10 w-80 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                  </div>
                </div>
              </div>
              <div className="flex w-[420px] h-full justify-end items-end">
                <form className="flex flex-row justify-end items-center gap-x-2 h-9 w-full">
                  <input type="hidden" />
                  <div className="h-9 min-w-64 items-end bg-grayA-3 animate-pulse transition-all border border-grayA-5 rounded-lg" />
                  <div className="variant-primary size-lg px-3.5 h-9 w-[62px] bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </form>
              </div>
            </div>
          </div>

          {/* IP Whitelist card skeleton - border="both" */}
          <div className="w-full">
            <form>
              <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-xl w-full h-[138px]">
                <div className="flex flex-col gap-1 text-sm w-fit">
                  <div className="font-medium text-accent-12 leading-5 tracking-normal">
                    <div className="flex items-center justify-start gap-2.5">
                      <span className="text-sm font-medium text-accent-12">
                        <div className="h-5 w-20 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                      </span>
                      <div className="h-4 w-12 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                    </div>
                  </div>
                  <div className="font-normal text-accent-11 text-[13px] leading-5 tracking-normal">
                    <div className="font-normal text-[13px]">
                      <div className="h-[60px] w-96 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                    </div>
                  </div>
                </div>
                <div className="flex w-full">
                  <div className="flex flex-row justify-end items-start w-full gap-x-2">
                    <div className="size-lg variant-primary px-3.5 h-9 w-48 bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                  </div>
                </div>
              </div>
            </form>
          </div>

          {/* Delete protection and delete API cards skeleton */}
          <div>
            {/* Delete protection card skeleton - border="top" */}
            <div className="px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-t-xl border-b h-[115px]">
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="font-medium text-accent-12 leading-5 tracking-normal">
                  <div className="inline-flex gap-2">
                    <span>
                      <div className="h-5 w-32 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                    </span>
                    <div className="h-4 w-16 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                  </div>
                </div>
                <div className="font-normal text-accent-11 text-[13px] leading-5 tracking-normal">
                  <div className="h-10 w-80 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                </div>
              </div>
              <div className="flex w-full">
                <div className="flex flex-row justify-end items-start w-full gap-x-2">
                  <div className="size-lg variant-primary px-3.5 h-[42px] w-48 bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                </div>
              </div>
            </div>
            {/* Delete API card skeleton - border="bottom" */}
            <div>
              <div className="px-6 py-6 flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row border border-grayA-4 rounded-b-xl border-t-0 w-full h-[115px]">
                <div className="flex flex-col gap-1 text-sm w-fit">
                  <div className="font-medium text-accent-12 leading-5 tracking-normal">
                    <div className="inline-flex gap-2">
                      <span>
                        <div className="h-5 w-20 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                      </span>
                      <div className="h-4 w-12 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                    </div>
                  </div>
                  <div className="font-normal text-accent-11 text-[13px] leading-5 tracking-normal">
                    <div className="font-normal text-[13px] max-w-[380px]">
                      <div className="h-10 w-96 bg-grayA-3 rounded-sm animate-pulse transition-all" />
                    </div>
                  </div>
                </div>
                <div className="flex w-full">
                  <div className="flex flex-row justify-end items-start w-full gap-x-2">
                    <div className="size-lg variant-primary px-3.5 h-[42px] w-48 bg-grayA-3 animate-pulse transition-all border border-grayA-4 rounded-lg" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
