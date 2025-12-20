import { trpc } from "@/lib/trpc/client";
import { ChartActivity2 } from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";
import { useRef, useState } from "react";

export const LastUsedCell = ({ identityId }: { identityId: string }) => {
  const { data, isLoading, isError } = trpc.identity.latestVerification.useQuery(
    {
      identityId,
    },
    {
      trpc: {
        context: {
          skipBatch: true,
        },
      },
    },
  );
  const badgeRef = useRef<HTMLDivElement>(null);
  const [showTooltip, setShowTooltip] = useState(false);

  return (
    <Badge
      ref={badgeRef}
      className="px-1.5 rounded-md flex gap-2 items-center max-w-min h-[22px] border-none cursor-pointer bg-grayA-3 text-grayA-11"
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
