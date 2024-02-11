import { Loading } from "@/components/dashboard/loading";

export default function () {
  // You can add any UI inside Loading, including a Skeleton.
  return (
    <div className="flex items-center justify-center w-full h-full">
      <Loading />
    </div>
  );
}
