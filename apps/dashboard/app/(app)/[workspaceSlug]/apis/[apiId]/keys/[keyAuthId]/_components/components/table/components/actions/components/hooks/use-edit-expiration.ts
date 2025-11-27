import { useTRPC } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

export const useEditExpiration = (onSuccess?: () => void) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const updateKeyExpiration = useMutation(trpc.key.update.expiration.mutationOptions({
    onSuccess(_, variables) {
      let description = "";
      if (variables.expiration?.enabled && variables.expiration.data) {
        description = `Your key ${
          variables.keyId
        } has been updated to expire on ${variables.expiration.data.toLocaleString()}`;
      } else {
        description = `Expiration has been disabled for key ${variables.keyId}`;
      }

      toast.success("Key Expiration Updated", {
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
        toast.error("Invalid Request", {
          description: err.message || "Please check your expiration settings and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We were unable to update expiration on this key. Please try again or contact support@unkey.dev",
        });
      } else {
        toast.error("Failed to Update Key Expiration", {
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
  }));
  return updateKeyExpiration;
};
