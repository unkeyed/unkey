import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div>
      <div className="flex flex-col mt-4 gap-8">
        <Skeleton className="h-[231px] bg-transparent border border-round border-border">
          <Skeleton className="w-[180px] h-6 mt-6 mx-6" />
          <Skeleton className="w-[384px] h-8 mt-8 mx-6 bg-transparent border border-round border-border" />
          <Skeleton className="w-56 h-3 mt-3 mx-6" />
          <Skeleton className="w-full h-[1px] mt-9" />
          <Skeleton className="w-14 h-8 mt-3 ml-auto mr-6" />
        </Skeleton>
        <Skeleton className="flex flex-col-2 h-[154px] w-full p-6 bg-transparent border border-round border-border">
          <div className="">
            <Skeleton className="w-[180px] h-6" />
            <Skeleton className="w-[410px] h-3 mt-3" />
            <Skeleton className="w-[500px] h-3 mt-3" />
          </div>
          <div className="ml-auto justify-end items-end">
            <Skeleton className="w-24 h-24 rounded-full mt-1" />
          </div>
        </Skeleton>
        <Skeleton className="h-40 w-full p-6 bg-transparent border border-round border-border">
          <Skeleton className="w-[150px] h-6" />
          <Skeleton className="w-[345px] h-3 mt-4" />
          <Skeleton className="w-[384px] h-8 mt-[26px] bg-transparent border border-round border-border" />
        </Skeleton>
      </div>
    </div>
  );
}
