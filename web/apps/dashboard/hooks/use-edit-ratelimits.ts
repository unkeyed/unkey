import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { formatDuration, intervalToDuration } from "date-fns";

type EntityType = "key" | "identity";

export function useEditRatelimits(
  entityType: "key",
  onSuccess?: () => void,
): ReturnType<typeof trpc.key.update.ratelimit.useMutation>;
export function useEditRatelimits(
  entityType: "identity",
  onSuccess?: () => void,
): ReturnType<typeof trpc.identity.update.ratelimit.useMutation>;
export function useEditRatelimits(entityType: EntityType, onSuccess?: () => void) {
  const trpcUtils = trpc.useUtils();

  const updateKeyRatelimit = trpc.key.update.ratelimit.useMutation({
    onSuccess(data, variables) {
      let description = "";

      if (variables.ratelimit?.enabled) {
        const rulesCount = variables.ratelimit.data.length;

        if (rulesCount === 1) {
          const rule = variables.ratelimit.data[0];
          const refillInterval = typeof rule.refillInterval === "number" ? rule.refillInterval : 0;
          description = `Your key ${data.keyId} has been updated with a limit of ${
            rule.limit
          } requests per ${formatInterval(refillInterval)}`;
        } else {
          description = `Your key ${data.keyId} has been updated with ${rulesCount} rate limit rules`;
        }
      } else {
        description = `Your key ${data.keyId} has been updated with rate limits disabled`;
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
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  const updateIdentityRatelimit = trpc.identity.update.ratelimit.useMutation({
    onSuccess(data, variables) {
      let description = "";

      if (variables.ratelimit?.enabled) {
        const rulesCount = variables.ratelimit.data.length;

        if (rulesCount === 1) {
          const rule = variables.ratelimit.data[0];
          const refillInterval = typeof rule.refillInterval === "number" ? rule.refillInterval : 0;
          description = `Identity ${data.identityId} has been updated with a limit of ${
            rule.limit
          } requests per ${formatInterval(refillInterval)}`;
        } else {
          description = `Identity ${data.identityId} has been updated with ${rulesCount} rate limit rules`;
        }
      } else {
        description = `Identity ${data.identityId} has been updated with rate limits disabled`;
      }

      toast.success("Identity Ratelimits Updated", {
        description,
        duration: 5000,
      });

      trpcUtils.identity.query.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Identity Update Failed", {
          description: "Unable to find the identity. Please refresh and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while updating your identity. Please try again later or contact support at support.unkey.dev",
        });
      } else {
        toast.error("Failed to Update Identity Limits", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  return entityType === "key" ? updateKeyRatelimit : updateIdentityRatelimit;
}

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
