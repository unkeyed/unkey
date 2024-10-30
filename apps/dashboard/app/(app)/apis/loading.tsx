import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div>
      <div className="flex flex-col pt-2">
        <Skeleton className="w-32 h-6" />
        <Skeleton className="w-28 h-3 mt-4" />
        <Skeleton className="w-full h-[1px] mt-6 md:mt-9 lg:mt-[52px]" />
      </div>
      <div className="flex max-sm:flex-col md:flex-row mt-6 gap-4">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="sm:basis-full md:basis-48 lg:basis-[168px] h-8" />
      </div>
      <div className="flex lg:flex-row flex-col mt-4 gap-6">
        <Skeleton className="h-44 w-full p-6 bg-transparent border border-round border-border">
          <Skeleton className="w-[130px] h-6" />
          <Skeleton className="w-[215px] h-3 mt-4" />
          <div className="w-full h-full mt-11 flex flex-row">
            <Skeleton className="w-14 h-3" />
            <Skeleton className="w-3 h-3 ml-auto" />
          </div>
        </Skeleton>
        <Skeleton className="h-44 w-full p-6 bg-transparent border border-round border-border">
          <Skeleton className="w-[130px] h-6" />
          <Skeleton className="w-[215px] h-3 mt-4" />
          <div className="w-full h-full mt-11 flex flex-row">
            <Skeleton className="w-14 h-3" />
            <Skeleton className="w-3 h-3 ml-auto" />
          </div>
        </Skeleton>
        <Skeleton className="h-44 w-full p-6 bg-transparent border border-round border-border">
          <Skeleton className="w-[130px] h-6" />
          <Skeleton className="w-[215px] h-3 mt-4" />
          <div className="w-full h-full mt-11 flex flex-row">
            <Skeleton className="w-14 h-3" />
            <Skeleton className="w-3 h-3 ml-auto" />
          </div>
        </Skeleton>
      </div>
    </div>
  );
}
