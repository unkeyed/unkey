import { useTRPC } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { cn } from "@/lib/utils";
import { InfoTooltip, toast } from "@unkey/ui";
import { StatusBadge } from "./components/status-badge";
import { useKeyStatus } from "./use-key-status";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

type StatusDisplayProps = {
  keyData: KeyDetails;
  keyAuthId: string;
  isSelected: boolean;
};

export const StatusDisplay = ({ keyAuthId, keyData, isSelected }: StatusDisplayProps) => {
  const trpc = useTRPC();
  const { primary, count, isLoading, statuses, isError } = useKeyStatus(keyAuthId, keyData);
  const queryClient = useQueryClient();

  const enableKeyMutation = useMutation(trpc.api.keys.enableKey.mutationOptions({
    onSuccess: async () => {
      toast.success("Key enabled successfully!");
      await queryClient.invalidateQueries(trpc.api.keys.list.queryFilter({ keyAuthId }));
    },
    onError: (error) => {
      toast.error("Failed to enable key", {
        description: error.message || "An unknown error occurred.",
      });
    },
  }));

  if (isLoading) {
    return (
      <div
        className="flex w-[100px] items-center h-[22px] space-x-1 px-1.5 py-1 rounded-md bg-grayA-3"
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
    <InfoTooltip
      position={{ side: "top", align: "center" }}
      disabled={false}
      content={
        <div>
          {statuses && statuses.length > 1 && (
            <div className="border-b border-grayA-3">
              <div className="flex flex-col px-[1px] py-[1px] gap-1 w-[260px] p-1">
                <div className="text-accent-12 font-medium text-[13px]">Key status overview</div>
                <div className="text-accent-10 text-xs ">
                  This key has{" "}
                  <span className="font-semibold text-accent-12">{statuses.length}</span> active
                  flags{" "}
                </div>
              </div>
            </div>
          )}

          {statuses?.map((status, i) => (
            <div
              className={cn("border-grayA-3", i !== statuses.length - 1 && "border-b")}
              key={status.type || i}
            >
              <div className="flex items-start gap-1.5 flex-col w-[260px] p-1">
                <div className="flex-shrink-0 mt-0.5">
                  <StatusBadge
                    primary={{
                      label: status.label,
                      color: status.color,
                      icon: status.icon,
                    }}
                    count={0}
                  />
                </div>

                <div className="text-xs text-accent-11 text-wrap leading-6 flex-grow w-full">
                  {status.type === "disabled" ? (
                    <div className="flex flex-col gap-2 items-start">
                      <span>
                        This key has been manually disabled and cannot be used for any requests.
                      </span>
                      <button
                        type="button"
                        onClick={() => {
                          if (enableKeyMutation.isPending) {
                            return;
                          }

                          if (keyData?.id) {
                            enableKeyMutation.mutate({ keyId: keyData.id });
                          } else {
                            toast.error("Could not enable key: Missing key information.");
                          }
                        }}
                        disabled={enableKeyMutation.isPending}
                        className={cn(
                          "bg-transparent border-none p-0 m-0 text-left",
                          "font-medium",
                          "text-xs",
                          "transition-colors duration-150 ease-in-out",
                          enableKeyMutation.isPending
                            ? "text-gray-10 cursor-not-allowed"
                            : "text-info-11 hover:text-info-12 hover:underline cursor-pointer",
                        )}
                      >
                        {enableKeyMutation.isPending ? "Enabling..." : "Re-enable this key"}
                      </button>{" "}
                    </div>
                  ) : (
                    status.tooltip
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      }
    >
      <StatusBadge primary={primary} count={count} isSelected={isSelected} />
    </InfoTooltip>
  );
};
