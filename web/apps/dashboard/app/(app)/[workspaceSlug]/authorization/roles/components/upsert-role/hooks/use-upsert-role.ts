import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useUpsertRole = (
  onSuccess: (data: {
    roleId?: string;
    isUpdate: boolean;
    message: string;
  }) => void,
) => {
  const trpcUtils = trpc.useUtils();

  const role = trpc.authorization.roles.upsert.useMutation({
    async onSuccess(data) {
      await Promise.all([
        trpcUtils.authorization.roles.query.invalidate(),
        trpcUtils.authorization.permissions.query.refetch(),
        trpcUtils.authorization.roles.connectedKeysAndPerms.invalidate({
          roleId: data.roleId,
        }),
        trpcUtils.authorization.roles.connectedKeys.invalidate({
          roleId: data.roleId,
        }),
        trpcUtils.authorization.roles.connectedPerms.invalidate({
          roleId: data.roleId,
        }),
      ]);

      // Show success toast
      toast.success(data.isUpdate ? "Role Updated" : "Role Created", {
        description: data.message,
      });

      onSuccess(data);
    },
    onError(err) {
      if (err.data?.code === "CONFLICT") {
        toast.error("Role Already Exists", {
          description: err.message || "A role with this name already exists in your workspace.",
        });
      } else if (err.data?.code === "NOT_FOUND") {
        toast.error("Role Not Found", {
          description:
            "The role you're trying to update no longer exists or you don't have access to it.",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Role Configuration", {
          description: `Please check your role settings. ${err.message || ""}`,
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while saving your role. Please try again later or contact support.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      } else {
        toast.error("Failed to Save Role", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  return role;
};
