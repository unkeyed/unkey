import { Skeleton } from "@/components/ui/skeleton";
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
export default async function Loading() {
  await sleep(3000);
  return (
    <div>
      <div className="grid grid-cols-2 max-sm:gap-2 gap-4 md:grid-cols-3">
        <Skeleton className="h-[104px] p-6 max-md:mr-2 max-md:mb-2 max-sm:mb-2 bg-transparent border border-round border-border">
          <Skeleton className="w-4 h-6" />
          <Skeleton className="w-20 h-3 mt-4" />
        </Skeleton>
        <Skeleton className="h-[104px] p-6 max-md:mb-4 max-sm:mb-2 bg-transparent border border-round border-border">
          <Skeleton className="w-4 h-6" />
          <Skeleton className="w-36 h-3 mt-4" />
        </Skeleton>

        <Skeleton className="h-[104px] p-6 max-md:mb-2 col-span-2 md:col-span-1 bg-transparent border border-round border-border">
          <Skeleton className="w-4 h-6" />
          <Skeleton className="w-56 h-3 mt-4" />
        </Skeleton>
      </div>

      <Skeleton className="h-[526px] bg-transparent border border-round border-border pb-10 max-sm:mt-2 mt-4">
        <Skeleton className="flex flex-col-2 pl-6 pt-7 bg-transparent  ">
          <div className="w-1/2 h-28 ">
            <Skeleton className="w-64 h-6" />
            <Skeleton className="w-72 h-3 mt-3" />
          </div>
          <div className="flex flex-row w-1/2 h-[117px] gap-8 p-6 justify-end items-end">
            <Skeleton className="w-36 h-2" />
            <Skeleton className="w-[88px] h-2" />
            <Skeleton className="w-28 h-2 mr-4" />
          </div>
        </Skeleton>
        <div className="flex flex-col-32 gap-6 items-end h-[300px] px-10 justify-end ">
          <Skeleton className="md:hidden w-8 h-2" />
          <Skeleton className="w-8 h-2" />
          <Skeleton className="md:hidden w-8 h-3" />
          <Skeleton className="w-8 h-4" />
          <Skeleton className="md:hidden w-8 h-6" />
          <Skeleton className="w-8 h-6" />
          <Skeleton className="md:hidden w-8 h-7" />
          <Skeleton className="w-8 h-7" />
          <Skeleton className="md:hidden w-8 h-8" />
          <Skeleton className="w-8 h-9" />
          <Skeleton className="md:hidden w-8 h-10" />
          <Skeleton className="w-8 h-11" />
          <Skeleton className="md:hidden w-8 h-14" />
          <Skeleton className="w-8 h-16" />
          <Skeleton className="md:hidden w-8 h-20" />
          <Skeleton className="w-8 h-24" />
          <Skeleton className="w-8 h-32" />
          <Skeleton className="md:hidden w-8 h-36" />
          <Skeleton className="w-8 h-40" />
          <Skeleton className="md:hidden w-8 h-48" />
          <Skeleton className="w-8 h-52" />
          <Skeleton className="w-8 h-60" />
        </div>
      </Skeleton>
    </div>
  );
}
