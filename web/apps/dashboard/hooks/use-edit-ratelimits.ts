import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { formatDuration, intervalToDuration } from "date-fns";

export function useEditKeyRatelimits(onSuccess?: () => void) {
  const trpcUtils = trpc.useUtils();

  return trpc.key.update.ratelimit.useMutation({
    onSuccess(data, variables) {
      let description = "";

      if (variables.ratelimit?.enabled) {
        const rulesCount = variables.ratelimit.data.length;

        if (rulesCount === 1) {
          const rule = variables.ratelimit.data[0];
          const refillInterval = typeof rule.refillInterval === "number" ? rule.refillInterval : 0;
          description = `Your key ${data.keyId} has been updated with a limit of ${rule.limit} requests per ${formatInterval(refillInterval)}`;
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
      onSuccess?.();
    },
    onError(err) {
      handleError(err.data?.code, err.message, "key");
    },
  });
}

export function useEditIdentityRatelimits(onSuccess?: () => void) {
  const trpcUtils = trpc.useUtils();

  return trpc.identity.update.ratelimit.useMutation({
    onSuccess(data, variables) {
      let description = "";

      if (variables.ratelimit?.enabled) {
        const rulesCount = variables.ratelimit.data.length;

        if (rulesCount === 1) {
          const rule = variables.ratelimit.data[0];
          const refillInterval = typeof rule.refillInterval === "number" ? rule.refillInterval : 0;
          description = `Identity ${data.identityId} has been updated with a limit of ${rule.limit} requests per ${formatInterval(refillInterval)}`;
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
      onSuccess?.();
    },
    onError(err) {
      handleError(err.data?.code, err.message, "identity");
    },
  });
}

function handleError(
  code: string | undefined,
  message: string | undefined,
  entity: "key" | "identity",
) {
  const label = entity === "key" ? "Key" : "Identity";
  const entityName = entity === "key" ? "key" : "identity";

  if (code === "NOT_FOUND") {
    toast.error(`${label} Update Failed`, {
      description: `Unable to find the ${entityName}. Please refresh and try again.`,
    });
  } else if (code === "INTERNAL_SERVER_ERROR") {
    toast.error("Server Error", {
      description: `We encountered an issue while updating your ${entityName}. Please try again later or contact support at support.unkey.dev`,
    });
  } else {
    toast.error(`Failed to Update ${label} Limits`, {
      description: message || "An unexpected error occurred. Please try again later.",
      action: {
        label: "Contact Support",
        onClick: () => window.open("mailto:support@unkey.com", "_blank"),
      },
    });
  }
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
