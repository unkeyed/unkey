import { toast } from "@/components/ui/toaster";

import { trpc } from "@/lib/trpc/client";

export const useUpdateKeyRemaining = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const updateKeyRemaining = trpc.key.update.remaining.useMutation({
    onSuccess(data, variables) {
      const remainingChange = variables.limitEnabled
        ? `with ${variables.remaining} uses remaining`
        : "with limits disabled";

      toast.success("Key Limits Updated", {
        description: `Your key ${data.keyId} has been updated successfully ${remainingChange}`,
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
          description: "Unable to find the key. Please refresh and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while updating your key. Please try again later or contact support at support.unkey.dev",
        });
      } else {
        toast.error("Failed to Update Key Limits", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("https://support.unkey.dev", "_blank"),
          },
        });
      }
    },
  });
  return updateKeyRemaining;
};
