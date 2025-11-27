import { useTRPC } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { ChartActivity2 } from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";
import { useRef, useState } from "react";
import { STATUS_STYLES } from "../_overview/components/table/utils/get-row-class";

import { useQuery } from "@tanstack/react-query";

type LastUsedCellProps = {
  namespaceId: string;
  identifier: string;
};

export const LastUsedCell = ({ namespaceId, identifier }: LastUsedCellProps) => {
  const trpc = useTRPC();
  const { data, isLoading, isError } = useQuery(
    trpc.ratelimit.namespace.queryRatelimitLastUsed.queryOptions({
      namespaceId,
      identifier,
    }),
  );
  const badgeRef = useRef<HTMLDivElement>(null);
  const [showTooltip, setShowTooltip] = useState(false);

  return (
    <Badge
      ref={badgeRef}
      className={cn(
        "px-1.5 rounded-md flex gap-2 items-center max-w-min h-[22px] border-none cursor-pointer",
        isError
          ? "bg-error-3 text-error-11 border border-error-5"
          : STATUS_STYLES.success.badge.default,
      )}
      onMouseOver={() => {
        setShowTooltip(true);
      }}
      onMouseLeave={() => {
        setShowTooltip(false);
      }}
    >
      <div>
        <ChartActivity2 iconSize="sm-regular" />
      </div>
      <div className="truncate">
        {isLoading ? (
          <div className="flex items-center w-full space-x-1">
            <div className="h-2 w-2 bg-grayA-5 rounded-full animate-pulse" />
            <div className="h-2 w-12 bg-grayA-5 rounded animate-pulse" />
            <div className="h-2 w-12 bg-grayA-5 rounded animate-pulse" />
          </div>
        ) : isError ? (
          "Failed to load"
        ) : data?.lastUsed ? (
          <TimestampInfo
            displayType="relative"
            value={data.lastUsed}
            className="truncate"
            triggerRef={badgeRef}
            open={showTooltip}
            onOpenChange={setShowTooltip}
          />
        ) : (
          "Never used"
        )}
      </div>
    </Badge>
  );
};
