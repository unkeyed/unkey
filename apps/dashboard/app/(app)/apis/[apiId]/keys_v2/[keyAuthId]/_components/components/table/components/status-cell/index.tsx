// src/components/keys/StatusDisplay.tsx (or similar)
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { StatusBadge } from "./components/status-badge";
import { useKeyStatus } from "./use-key-status";

interface StatusDisplayProps {
  keyData: KeyDetails;
  keyAuthId: string;
}

export const StatusDisplay: React.FC<StatusDisplayProps> = ({ keyAuthId, keyData }) => {
  const { primary, count, tooltips, isLoading, isError } = useKeyStatus(keyAuthId, keyData);

  if (isLoading) {
    return (
      <div
        className="flex w-[100px] items-center h-[22px] space-x-1 px-1.5 py-1 rounded-md bg-gray-3"
        aria-busy="true"
        aria-live="polite"
      >
        <div className="h-2 w-2 bg-grayA-5 rounded-full animate-pulse" />
        <div className="h-2 w-20 bg-grayA-5 rounded animate-pulse" />
      </div>
    );
  }

  if (isError) {
    return (
      <div
        className={cn(
          "flex items-center justify-center h-[22px] w-[100px]",
          "px-1.5 py-1 rounded-md",
          "bg-errorA-3 text-errorA-11 text-xs",
        )}
        role="alert"
      >
        Failed to load
      </div>
    );
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div>
            <StatusBadge primary={primary} count={count} />
          </div>
        </TooltipTrigger>
        <TooltipContent>
          {tooltips?.map((tooltip, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
            <p key={i} className={i > 0 ? "mt-2" : ""}>
              {tooltip}
            </p>
          ))}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
