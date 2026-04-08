import { Dots, GripDotsVertical } from "@unkey/icons";
import { Button } from "@unkey/ui";

/**
 * Loading skeleton for SentinelPoliciesList. Mirrors the row layout in
 * row.tsx exactly so the list doesn't shift when real data lands.
 */
export function SentinelPoliciesListSkeleton({ rows = 10 }: { rows?: number }) {
  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
      <div>
        {Array.from({ length: rows }).map((_, i) => (
          <SentinelPolicyRowSkeleton key={i} index={i} isLast={i === rows - 1} />
        ))}
      </div>
    </div>
  );
}

function SentinelPolicyRowSkeleton({ index, isLast }: { index: number; isLast: boolean }) {
  return (
    <div className={isLast ? undefined : "border-b border-grayA-4"}>
      <div className="flex items-center">
        {/* Step number */}
        <div className="w-10 shrink-0 py-5 pl-4 flex items-center">
          <div className="size-6 rounded-full border bg-grayA-2 border-grayA-5 text-gray-10 flex items-center justify-center text-[11px] font-medium">
            {index + 1}
          </div>
        </div>

        {/* Drag handle */}
        <div className="w-10 shrink-0 flex items-center justify-center py-5">
          <GripDotsVertical iconSize="lg-medium" className="opacity-20" />
        </div>

        {/* Name */}
        <div className="flex-4 min-w-0 py-5 flex items-center pr-5">
          <div className="h-[13px] w-32 bg-grayA-3 rounded-sm animate-pulse" />
        </div>

        {/* Type */}
        <div className="flex-4 min-w-0 py-5 flex items-center pr-3">
          <div className="h-[13px] w-16 bg-grayA-3 rounded-sm animate-pulse" />
        </div>

        {/* Env badges */}
        <div className="flex-2 min-w-0 py-5 flex items-center gap-1.5 pr-3">
          <div className="h-[22px] w-full rounded-full border border-dashed border-grayA-4 bg-grayA-2 animate-pulse" />
          <div className="h-[22px] w-full rounded-full border border-dashed border-grayA-4 bg-grayA-2 animate-pulse" />
        </div>

        {/* Actions */}
        <div className="w-12 shrink-0 py-5 flex items-center justify-end pr-4">
          <Button
            variant="outline"
            className="size-5 [&_svg]:size-3 rounded-sm border-transparent"
            disabled
          >
            <Dots className="text-gray-11 opacity-30" iconSize="sm-regular" />
          </Button>
        </div>
      </div>
    </div>
  );
}
