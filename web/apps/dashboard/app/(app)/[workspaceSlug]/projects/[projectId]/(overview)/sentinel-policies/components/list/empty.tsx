import { Empty } from "@unkey/ui";

export function SentinelPoliciesEmpty() {
  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
      <div className="flex items-center justify-center py-16 px-4">
        <Empty className="w-[400px] flex items-start">
          <Empty.Icon className="w-auto" />
          <Empty.Title>No Sentinel Policies</Empty.Title>
          <Empty.Description className="text-left">
            Add policies to protect your API with authentication, rate limiting, and more. Policies
            are evaluated sequentially on each incoming request.
          </Empty.Description>
        </Empty>
      </div>
    </div>
  );
}
