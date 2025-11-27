import { useTRPC } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

export const useDeleteRootKey = (
  onSuccess: (data: { keyIds: string[]; message: string }) => void,
) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const deleteRootKey = useMutation(
    trpc.settings.rootKeys.delete.mutationOptions({
      onSuccess(_, variables) {
        queryClient.invalidateQueries(trpc.settings.rootKeys.query.pathFilter());
        toast.success("Root Key Deleted", {
          description:
            "The root key has been permanently deleted and can no longer create resources.",
        });
        onSuccess({
          keyIds: Array.isArray(variables.keyIds) ? variables.keyIds : [variables.keyIds],
          message: "Root key deleted successfully",
        });
      },
      onError(err) {
        if (err.data?.code === "NOT_FOUND") {
          toast.error("Root Key Not Found", {
            description:
              "The root key you're trying to revoke no longer exists or you don't have access to it.",
          });
        } else if (err.data?.code === "BAD_REQUEST") {
          toast.error("Invalid Request", {
            description: err.message || "Please provide a valid root key to revoke.",
          });
        } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
          toast.error("Server Error", {
            description:
              "We encountered an issue while revoking your root key. Please try again later or contact support.",
            action: {
              label: "Contact Support",
              onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
            },
          });
        } else {
          toast.error("Failed to Revoke Root Key", {
            description: err.message || "An unexpected error occurred. Please try again later.",
            action: {
              label: "Contact Support",
              onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
            },
          });
        }
      },
    }),
  );
  return deleteRootKey;
};
