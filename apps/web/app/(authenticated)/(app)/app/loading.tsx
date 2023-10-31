import { Loading } from "@/components/dashboard/loading";
export default function LoadingPage() {
  // You can add any UI inside Loading, including a Skeleton.
  return (
    <div className="h-screen flex items-center justify-center">
      <h1 className="font-12xl">Loading</h1>
      <Loading width="48" height="48" />
    </div>
  );
}
