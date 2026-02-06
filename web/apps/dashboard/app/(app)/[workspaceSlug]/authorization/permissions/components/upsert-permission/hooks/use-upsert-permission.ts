import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useUpsertPermission = (
  onSuccess: (data: {
    permissionId?: string;
    isUpdate: boolean;
    message: string;
  }) => void,
) => {
  const trpcUtils = trpc.useUtils();
  const permission = trpc.authorization.permissions.upsert.useMutation({
    onSuccess(data) {
      trpcUtils.authorization.permissions.invalidate();
      trpcUtils.authorization.roles.invalidate();
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
            err.message || "A permission with this name or slug already exists in your workspace.",
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
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      } else {
        toast.error("Failed to Save Permission", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });
  return permission;
};
