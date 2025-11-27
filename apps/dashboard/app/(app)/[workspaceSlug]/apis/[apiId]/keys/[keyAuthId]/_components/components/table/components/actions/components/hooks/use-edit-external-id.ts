import { useTRPC } from "@/lib/trpc/client";
import type { TRPCClientErrorLike } from "@trpc/client";
import { toast } from "@unkey/ui";

import type { AppRouter } from "@/lib/trpc/routers";
import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

const handleKeyOwnerUpdateError = (err: TRPCClientErrorLike<AppRouter>) => {
  if (err.data?.code === "NOT_FOUND") {
    toast.error("Key Update Failed", {
      description: "Unable to find the key(s). Please refresh and try again.",
    });
  } else if (err.data?.code === "BAD_REQUEST") {
    toast.error("Invalid External ID Information", {
      description:
        err.message || "Please ensure your External ID information is valid and try again.",
    });
  } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
    toast.error("Server Error", {
      description:
        "We are unable to update External ID information on this key. Please try again or contact support@unkey.dev",
    });
  } else {
    toast.error("Failed to Update Key External ID", {
      description:
        err.message ||
        "An unexpected error occurred. Please try again or contact support@unkey.dev",
      action: {
        label: "Contact Support",
        onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
      },
    });
  }
};

export const useEditExternalId = (onSuccess?: () => void) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const updateKeyOwner = useMutation(
    trpc.key.update.ownerId.mutationOptions({
      onSuccess(_, variables) {
        let description = "";
        const keyId = Array.isArray(variables.keyIds) ? variables.keyIds[0] : variables.keyIds;

        if (variables.ownerType === "v2") {
          if (variables.identity?.id) {
            description = `Identity for key ${keyId} has been updated`;
          } else {
            description = `Identity has been removed from key ${keyId}`;
          }
        }
        toast.success("Key External ID Updated", {
          description,
          duration: 5000,
        });

        queryClient.invalidateQueries(trpc.api.keys.list.pathFilter());
        if (onSuccess) {
          onSuccess();
        }
      },
      onError(err) {
        handleKeyOwnerUpdateError(err);
      },
    }),
  );

  return updateKeyOwner;
};

export const useBatchEditExternalId = (onSuccess?: () => void) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const batchUpdateKeyOwner = useMutation(
    trpc.key.update.ownerId.mutationOptions({
      onSuccess(data, variables) {
        const updatedCount = data.updatedCount;
        let description = "";

        if (variables.ownerType === "v2") {
          if (variables.identity?.id) {
            description = `Identity has been updated for ${updatedCount} ${
              updatedCount === 1 ? "key" : "keys"
            }`;
          } else {
            description = `Identity has been removed from ${updatedCount} ${
              updatedCount === 1 ? "key" : "keys"
            }`;
          }
        }
        toast.success("Key External ID Updated", {
          description,
          duration: 5000,
        });

        // Show warning if some keys were not found (if that info is available in the response)
        const missingCount = Array.isArray(variables.keyIds)
          ? variables.keyIds.length - updatedCount
          : 0;

        if (missingCount > 0) {
          toast.warning("Some Keys Not Found", {
            description: `${missingCount} ${
              missingCount === 1 ? "key was" : "keys were"
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
        handleKeyOwnerUpdateError(err);
      },
    }),
  );

  return batchUpdateKeyOwner;
};
