import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { formatDuration, intervalToDuration } from "date-fns";

export const useEditRatelimits = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();

  const updateKeyRemaining = trpc.key.update.ratelimit.useMutation({
    onSuccess(data, variables) {
      let description = "";

      // Handle both V1 and V2 ratelimit types
      if (variables.ratelimitType === "v2") {
        if (variables.ratelimit?.enabled) {
          const rulesCount = variables.ratelimit.data.length;

          if (rulesCount === 1) {
            // If there's just one rule, show its limit directly
            const rule = variables.ratelimit.data[0];
            description = `Your key ${data.keyId} has been updated with a limit of ${
              rule.limit
            } requests per ${formatInterval(rule.refillInterval)}`;
          } else {
            // If there are multiple rules, show the count
            description = `Your key ${data.keyId} has been updated with ${rulesCount} rate limit rules`;
          }
        } else {
          description = `Your key ${data.keyId} has been updated with rate limits disabled`;
        }
      } else {
        // V1 ratelimits
        if (variables.enabled) {
          description = `Your key ${data.keyId} has been updated with a limit of ${
            variables.ratelimitLimit
          } requests per ${formatInterval(variables.ratelimitDuration || 0)}`;
        } else {
          description = `Your key ${data.keyId} has been updated with rate limits disabled`;
        }
      }

      toast.success("Key Ratelimits Updated", {
        description,
        duration: 5000,
      });

      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Update Failed", {
          description: "Unable to find the key. Please refresh and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while updating your key. Please try again later or contact support at support.unkey.dev",
        });
      } else {
        toast.error("Failed to Update Key Limits", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("https://support.unkey.dev", "_blank"),
          },
        });
      }
    },
  });

  return updateKeyRemaining;
};

const formatInterval = (milliseconds: number): string => {
  if (milliseconds < 1000) {
    return `${milliseconds}ms`;
  }

  const duration = intervalToDuration({ start: 0, end: milliseconds });

  // Customize the format for different time ranges
  if (milliseconds < 60000) {
    // Less than a minute
    return formatDuration(duration, { format: ["seconds"] });
  }
  if (milliseconds < 3600000) {
    // Less than an hour
    return formatDuration(duration, { format: ["minutes", "seconds"] });
  }
  if (milliseconds < 86400000) {
    // Less than a day
    return formatDuration(duration, { format: ["hours", "minutes"] });
  }
  // Days or more
  return formatDuration(duration, { format: ["days", "hours"] });
};
