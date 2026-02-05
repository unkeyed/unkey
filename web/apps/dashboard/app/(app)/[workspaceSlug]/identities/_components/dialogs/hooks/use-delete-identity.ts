import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useDeleteIdentity = (onSuccess: () => void) => {
  const trpcUtils = trpc.useUtils();

  const deleteIdentity = trpc.identity.delete.useMutation({
    onSuccess() {
      toast.success("Identity Deleted", {
        description:
          "The identity has been permanently deleted and can no longer be used for verification.",
      });

      trpcUtils.identity.query.invalidate();
      trpcUtils.identity.search.invalidate();
      trpcUtils.identity.searchWithRelations.invalidate();

      onSuccess();
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Identity Not Found", {
          description:
            "The identity you're trying to delete no longer exists or you don't have access to it.",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Request", {
          description: err.message || "Please provide a valid identity to delete.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while deleting your identity. Please try again later or contact support.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      } else {
        toast.error("Failed to Delete Identity", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  return deleteIdentity;
};
