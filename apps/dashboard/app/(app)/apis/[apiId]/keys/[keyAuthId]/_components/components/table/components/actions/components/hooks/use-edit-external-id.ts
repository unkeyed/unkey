import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";

export const useEditExternalId = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();

  const updateKeyOwner = trpc.key.update.ownerId.useMutation({
    onSuccess(_, variables) {
      let description = "";

      if (variables.ownerType === "v2") {
        if (variables.identity?.id) {
          description = `Identity for key ${variables.keyId} has been updated`;
        } else {
          description = `Identity has been removed from key ${variables.keyId}`;
        }
      } else {
        if (variables.ownerId) {
          description = `Owner for key ${variables.keyId} has been updated`;
        } else {
          description = `Owner has been removed from key ${variables.keyId}`;
        }
      }

      toast.success("Key Owner Updated", {
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
        toast.error("Invalid Owner Information", {
          description:
            err.message || "Please ensure your owner information is valid and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We are unable to update owner information on this key. Please try again or contact support@unkey.dev",
        });
      } else {
        toast.error("Failed to Update Key Owner", {
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

  return updateKeyOwner;
};
