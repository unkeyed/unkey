import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useEditIdentityMetadata = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  return trpc.identity.update.metadata.useMutation({
    onSuccess(_, variables) {
      const description = variables.metadata?.enabled
        ? `Metadata for identity ${variables.identityId} has been updated`
        : `Metadata has been removed from identity ${variables.identityId}`;

      toast.success("Identity Metadata Updated", {
        description,
        duration: 5000,
      });
      trpcUtils.identity.query.invalidate();
      trpcUtils.identity.getById.invalidate();
      onSuccess?.();
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Identity Update Failed", {
          description:
            "We are unable to find the correct identity. Please try again or contact support@unkey.com.",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Metadata", {
          description: err.message || "Please ensure your metadata is valid JSON and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We are unable to update metadata on this identity. Please try again or contact support@unkey.com",
        });
      } else {
        toast.error("Failed to Update Identity Metadata", {
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
};
