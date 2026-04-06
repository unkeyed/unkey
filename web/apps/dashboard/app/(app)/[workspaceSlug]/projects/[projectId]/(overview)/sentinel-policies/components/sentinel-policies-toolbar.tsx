"use client";

type SentinelPoliciesToolbarProps = {
  policyCount: number;
  envASlug: string;
  envBSlug: string;
};

export function SentinelPoliciesToolbar({
  policyCount,
  envASlug,
  envBSlug,
}: SentinelPoliciesToolbarProps) {
  return (
    <div className="flex items-center justify-between w-full">
      <span className="text-sm text-gray-11">
        {policyCount} {policyCount === 1 ? "policy" : "policies"}
      </span>
      <div className="flex items-center gap-2">
        <span className="text-xs text-gray-10 px-2 py-0.5 rounded-md border border-grayA-4 bg-grayA-2 capitalize">
          {envASlug}
        </span>
        <span className="text-xs text-gray-10 px-2 py-0.5 rounded-md border border-grayA-4 bg-grayA-2 capitalize">
          {envBSlug}
        </span>
      </div>
    </div>
  );
}
