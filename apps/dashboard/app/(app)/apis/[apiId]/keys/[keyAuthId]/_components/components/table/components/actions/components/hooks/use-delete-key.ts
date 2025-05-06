import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";

export const useDeleteKey = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();

  const deleteKey = trpc.key.delete.useMutation({
    onSuccess() {
      toast.success("Key Deleted", {
        description: "Your key has been permanently deleted successfully",
        duration: 5000,
      });

      trpcUtils.api.keys.list.invalidate();

      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      const errorMessage = err.message || "";

      if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Deletion Failed", {
          description: "Unable to find the key. Please refresh and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while deleting your key. Please try again later or contact support at support.unkey.dev",
        });
      } else if (err.data?.code === "FORBIDDEN") {
        toast.error("Permission Denied", {
          description: "You don't have permission to delete this key.",
        });
      } else {
        toast.error("Failed to Delete Key", {
          description: errorMessage || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("https://support.unkey.dev", "_blank"),
          },
        });
      }
    },
  });

  return deleteKey;
};
