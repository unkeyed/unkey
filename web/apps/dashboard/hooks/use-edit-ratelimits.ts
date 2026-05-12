import { formatMs } from "@/lib/ms";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

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
          description = `Your key ${data.keyId} has been updated with a limit of ${rule.limit} requests per ${formatMs(refillInterval, { long: true })}`;
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
          description = `Identity ${data.identityId} has been updated with a limit of ${rule.limit} requests per ${formatMs(refillInterval, { long: true })}`;
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
      trpcUtils.identity.getById.invalidate({ identityId: data.identityId });
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
