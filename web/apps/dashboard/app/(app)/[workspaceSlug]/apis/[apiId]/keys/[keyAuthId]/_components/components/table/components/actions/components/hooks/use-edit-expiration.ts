import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useEditExpiration = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const updateKeyExpiration = trpc.key.update.expiration.useMutation({
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
      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Update Failed", {
          description:
            "We are unable to find the correct key. Please try again or contact support@unkey.com.",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Request", {
          description: err.message || "Please check your expiration settings and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We were unable to update expiration on this key. Please try again or contact support@unkey.com",
        });
      } else {
        toast.error("Failed to Update Key Expiration", {
          description:
            err.message ||
            "An unexpected error occurred. Please try again or contact support@unkey.com",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });
  return updateKeyExpiration;
};
