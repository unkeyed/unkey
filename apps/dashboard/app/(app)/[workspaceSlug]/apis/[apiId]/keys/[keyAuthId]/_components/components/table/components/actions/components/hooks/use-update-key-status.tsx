import { useTRPC } from "@/lib/trpc/client";
import type { TRPCClientErrorLike } from "@trpc/client";
import { toast } from "@unkey/ui";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";
import type { DefaultErrorShape } from "@trpc/server/unstable-core-do-not-import";

const handleKeyUpdateError = (
  err: TRPCClientErrorLike<{
    input: {
      enabled: boolean;
      keyIds: string | string[];
    };
    output: {
      enabled: boolean;
      updatedKeyIds: string[];
      missingKeyIds: string[] | undefined;
    };
    transformer: true;
    errorShape: DefaultErrorShape;
  }>,
) => {
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
        onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
      },
    });
  }
};

export const useUpdateKeyStatus = (onSuccess?: () => void) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const updateKeyEnabled = useMutation(
    trpc.key.update.enabled.mutationOptions({
      onSuccess(data) {
        toast.success(`Key ${data.enabled ? "Enabled" : "Disabled"}`, {
          description: `Your key ${data.updatedKeyIds[0]} has been ${
            data.enabled ? "enabled" : "disabled"
          } successfully`,
          duration: 5000,
        });
        queryClient.invalidateQueries(trpc.api.keys.list.pathFilter());
        if (onSuccess) {
          onSuccess();
        }
      },
      onError(err) {
        handleKeyUpdateError(err);
      },
    }),
  );

  return updateKeyEnabled;
};

export const useBatchUpdateKeyStatus = (onSuccess?: () => void) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const updateMultipleKeysEnabled = useMutation(
    trpc.key.update.enabled.mutationOptions({
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

        queryClient.invalidateQueries(trpc.api.keys.list.pathFilter());
        if (onSuccess) {
          onSuccess();
        }
      },
      onError(err) {
        handleKeyUpdateError(err);
      },
    }),
  );

  return updateMultipleKeysEnabled;
};
