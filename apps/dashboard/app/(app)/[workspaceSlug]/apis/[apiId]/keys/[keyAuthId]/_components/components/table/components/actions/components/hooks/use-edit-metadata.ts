import { useTRPC } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

export const useEditMetadata = (onSuccess?: () => void) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const updateKeyMetadata = useMutation(
    trpc.key.update.metadata.mutationOptions({
      onSuccess(_, variables) {
        let description = "";
        if (variables.metadata?.enabled && variables.metadata.data) {
          description = `Metadata for key ${variables.keyId} has been updated`;
        } else {
          description = `Metadata has been removed from key ${variables.keyId}`;
        }

        toast.success("Key Metadata Updated", {
          description,
          duration: 5000,
        });
        queryClient.invalidateQueries(trpc.api.keys.list.pathFilter());
        if (onSuccess) {
          onSuccess();
        }
      },
      onError(err) {
        if (err.data?.code === "NOT_FOUND") {
          toast.error("Key Update Failed", {
            description:
              "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
          });
        } else if (err.data?.code === "BAD_REQUEST") {
          toast.error("Invalid Metadata", {
            description: err.message || "Please ensure your metadata is valid JSON and try again.",
          });
        } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
          toast.error("Server Error", {
            description:
              "We are unable to update metadata on this key. Please try again or contact support@unkey.dev",
          });
        } else {
          toast.error("Failed to Update Key Metadata", {
            description:
              err.message ||
              "An unexpected error occurred. Please try again or contact support@unkey.dev",
            action: {
              label: "Contact Support",
              onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
            },
          });
        }
      },
    }),
  );
  return updateKeyMetadata;
};
