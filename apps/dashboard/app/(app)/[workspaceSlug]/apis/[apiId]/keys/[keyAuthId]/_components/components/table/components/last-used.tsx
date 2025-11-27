import { useTRPC } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { ChartActivity2 } from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";
import { useRef, useState } from "react";
import { STATUS_STYLES } from "../utils/get-row-class";

import { useQuery } from "@tanstack/react-query";

export const LastUsedCell = ({
  keyAuthId,
  keyId,
  isSelected,
}: {
  keyAuthId: string;
  keyId: string;
  isSelected: boolean;
}) => {
  const trpc = useTRPC();
  const { data, isLoading, isError } = useQuery(
    trpc.api.keys.latestVerification.queryOptions(
      {
        keyAuthId,
        keyId,
      },
      {
        trpc: {
          context: {
            skipBatch: true,
          },
        },
      },
    ),
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
          : isSelected
            ? STATUS_STYLES.badge.selected
            : STATUS_STYLES.badge.default,
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
        ) : data?.lastVerificationTime ? (
          <TimestampInfo
            displayType="relative"
            value={data.lastVerificationTime}
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
