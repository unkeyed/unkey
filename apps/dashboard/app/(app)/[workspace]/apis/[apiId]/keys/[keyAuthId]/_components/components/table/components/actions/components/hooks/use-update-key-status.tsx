import { trpc } from "@/lib/trpc/client";
import type { TRPCClientErrorLike } from "@trpc/client";
import type { TRPCErrorShape } from "@trpc/server/rpc";
import { toast } from "@unkey/ui";

const handleKeyUpdateError = (err: TRPCClientErrorLike<TRPCErrorShape>) => {
  const errorMessage = err.message || "";
  if (err.data?.code === "NOT_FOUND") {
    toast.error("Key Update Failed", {
      description: "Unable to find the key(s). Please refresh and try again.",
    });
  } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
    toast.error("Server Error", {
      description:
        "We encountered an issue while updating your key(s). Please try again later or contact support at support.unkey.dev",
    });
  } else {
    toast.error("Failed to Update Key Status", {
      description: errorMessage || "An unexpected error occurred. Please try again later.",
      action: {
        label: "Contact Support",
        onClick: () => window.open("https://support.unkey.dev", "_blank"),
      },
    });
  }
};

export const useUpdateKeyStatus = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();

  const updateKeyEnabled = trpc.key.update.enabled.useMutation({
    onSuccess(data) {
      toast.success(`Key ${data.enabled ? "Enabled" : "Disabled"}`, {
        description: `Your key ${data.updatedKeyIds[0]} has been ${
          data.enabled ? "enabled" : "disabled"
        } successfully`,
        duration: 5000,
      });
      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      handleKeyUpdateError(err);
    },
  });

  return updateKeyEnabled;
};

export const useBatchUpdateKeyStatus = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();

  const updateMultipleKeysEnabled = trpc.key.update.enabled.useMutation({
    onSuccess(data) {
      const updatedCount = data.updatedKeyIds.length;
      toast.success(`Keys ${data.enabled ? "Enabled" : "Disabled"}`, {
        description: `${updatedCount} ${
          updatedCount === 1 ? "key has" : "keys have"
        } been ${data.enabled ? "enabled" : "disabled"} successfully`,
        duration: 5000,
      });

      // Show warning if some keys were not found
      if (data.missingKeyIds && data.missingKeyIds.length > 0) {
        toast.warning("Some Keys Not Found", {
          description: `${data.missingKeyIds.length} ${
            data.missingKeyIds.length === 1 ? "key was" : "keys were"
          } not found and could not be updated.`,
          duration: 7000,
        });
      }

      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      handleKeyUpdateError(err);
    },
  });

  return updateMultipleKeysEnabled;
};
