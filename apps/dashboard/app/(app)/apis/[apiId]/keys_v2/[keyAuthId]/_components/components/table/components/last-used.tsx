import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { ChartActivity2 } from "@unkey/icons";
import { STATUS_STYLES } from "../utils/get-row-class";

export const LastUsedCell = ({
  keyAuthId,
  keyId,
  isSelected,
}: {
  keyAuthId: string;
  keyId: string;
  isSelected: boolean;
}) => {
  const { data, isLoading, isError } = trpc.api.keys.latestVerification.useQuery({
    keyAuthId,
    keyId,
  });

  return (
    <Badge
      className={cn(
        "px-1.5 rounded-md flex gap-2 items-center w-[140px]",
        isError
          ? "bg-error-3 text-error-11 border border-error-5"
          : isSelected
            ? STATUS_STYLES.badge.selected
            : STATUS_STYLES.badge.default,
      )}
    >
      <div>
        <ChartActivity2 size="sm-regular" />
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
          <TimestampInfo value={data.lastVerificationTime} className="truncate" />
        ) : (
          "Never used"
        )}
      </div>
    </Badge>
  );
};
