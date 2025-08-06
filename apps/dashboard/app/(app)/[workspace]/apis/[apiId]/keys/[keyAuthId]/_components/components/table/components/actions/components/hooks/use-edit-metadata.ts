import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useEditMetadata = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const updateKeyMetadata = trpc.key.update.metadata.useMutation({
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
      trpcUtils.api.keys.list.invalidate();
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
  });
  return updateKeyMetadata;
};
