import { Dots } from "@unkey/icons";

export function EnvVarsSkeleton() {
  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
      {Array.from({ length: 10 }).map((_, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
        <div key={i} className="flex items-center h-[76px]">
          <div className="flex-[3] min-w-0 py-3.5 flex items-center">
            <div className="flex items-center gap-3 px-4 w-full">
              <div className="flex flex-col gap-2 min-w-0 flex-1">
                <div className="h-[14px] w-32 bg-grayA-3 rounded-sm animate-pulse" />
                <div className="h-3 w-20 bg-grayA-3 rounded-sm animate-pulse" />
              </div>
            </div>
          </div>
          <div className="flex-[4] min-w-0 py-3.5 flex items-center">
            <div className="flex items-center gap-2">
              <div className="size-5 bg-grayA-3 rounded-md animate-pulse shrink-0" />
              <div className="h-[14px] w-24 bg-grayA-3 rounded-sm animate-pulse" />
            </div>
          </div>
          <div className="flex-[2] min-w-0 py-3.5 flex items-center pr-3">
            <div className="h-[22px] w-16 bg-grayA-3 rounded-md animate-pulse" />
          </div>
          <div className="w-12 shrink-0 py-3.5 pr-3 flex items-center justify-end">
            <Dots iconSize="sm-regular" className="text-gray-11 opacity-30" />
          </div>
        </div>
      ))}
    </div>
  );
}
