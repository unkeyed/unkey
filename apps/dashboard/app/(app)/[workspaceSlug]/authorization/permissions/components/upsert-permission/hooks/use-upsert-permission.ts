import { useTRPC } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

export const useUpsertPermission = (
  onSuccess: (data: {
    permissionId?: string;
    isUpdate: boolean;
    message: string;
  }) => void,
) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const permission = useMutation(
    trpc.authorization.permissions.upsert.mutationOptions({
      onSuccess(data) {
        queryClient.invalidateQueries(trpc.authorization.permissions.pathFilter());
        queryClient.invalidateQueries(trpc.authorization.roles.pathFilter());
        // Show success toast
        toast.success(data.isUpdate ? "Permission Updated" : "Permission Created", {
          description: data.message,
        });
        onSuccess(data);
      },
      onError(err) {
        if (err.data?.code === "CONFLICT") {
          toast.error("Permission Already Exists", {
            description:
              err.message ||
              "A permission with this name or slug already exists in your workspace.",
          });
        } else if (err.data?.code === "NOT_FOUND") {
          toast.error("Permission Not Found", {
            description:
              "The permission you're trying to update no longer exists or you don't have access to it.",
          });
        } else if (err.data?.code === "BAD_REQUEST") {
          toast.error("Invalid Permission Configuration", {
            description: `Please check your permission settings. ${err.message || ""}`,
          });
        } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
          toast.error("Server Error", {
            description:
              "We encountered an issue while saving your permission. Please try again later or contact support.",
            action: {
              label: "Contact Support",
              onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
            },
          });
        } else {
          toast.error("Failed to Save Permission", {
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
  return permission;
};
