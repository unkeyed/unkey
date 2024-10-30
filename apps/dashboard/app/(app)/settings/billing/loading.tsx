import { Loading } from "@/components/dashboard/loading";

export default function () {
  // You can add any UI inside Loading, including a Skeleton.
  return (
    <div className="w-full h-screen flex items-center justify-center">
      <Loading />
    </div>
  );
}
