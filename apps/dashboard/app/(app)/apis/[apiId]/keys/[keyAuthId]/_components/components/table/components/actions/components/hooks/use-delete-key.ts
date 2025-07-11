import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useDeleteKey = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const deleteKey = trpc.key.delete.useMutation({
    onSuccess(data, variable) {
      const deletedCount = data.totalDeleted;

      if (deletedCount === 1) {
        toast.success("Key Deleted", {
          description: "Your key has been permanently deleted successfully",
          duration: 5000,
        });
      } else {
        toast.success("Keys Deleted", {
          description: `${deletedCount} keys have been permanently deleted successfully`,
          duration: 5000,
        });
      }

      // If some keys weren't found. Someone might've already deleted them when this is fired.
      if (data.deletedKeyIds.length < variable.keyIds.length) {
        const missingCount = variable.keyIds.length - data.deletedKeyIds.length;
        toast.warning("Some Keys Not Found", {
          description: `${missingCount} ${
            missingCount === 1 ? "key was" : "keys were"
          } not found and could not be deleted.`,
          duration: 7000,
        });
      }

      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err, variable) {
      const errorMessage = err.message || "";
      const isPlural = variable.keyIds.length > 1;
      const keyText = isPlural ? "keys" : "key";

      if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Deletion Failed", {
          description: `Unable to find the ${keyText}. Please refresh and try again.`,
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description: `We encountered an issue while deleting your ${keyText}. Please try again later or contact support at support.unkey.dev`,
        });
      } else if (err.data?.code === "FORBIDDEN") {
        toast.error("Permission Denied", {
          description: `You don't have permission to delete ${
            isPlural ? "these keys" : "this key"
          }.`,
        });
      } else {
        toast.error(`Failed to Delete ${isPlural ? "Keys" : "Key"}`, {
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
