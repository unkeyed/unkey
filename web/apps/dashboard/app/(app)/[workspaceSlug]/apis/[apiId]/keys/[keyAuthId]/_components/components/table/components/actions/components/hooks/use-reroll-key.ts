import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useRerollKey = (onSuccess?: (data: { keyId: string; key: string }) => void) => {
  const trpcUtils = trpc.useUtils();
  const rerollKey = trpc.key.reroll.useMutation({
    onSuccess(data) {
      if (onSuccess) {
        onSuccess(data);
      }
    },
    onError(err) {
      const errorMessage = err.message || "";

      if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Not Found", {
          description: "Unable to find the key. Please refresh and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while rerolling your key. Please try again later or contact support at support.unkey.dev",
        });
      } else if (err.data?.code === "FORBIDDEN") {
        toast.error("Permission Denied", {
          description: "You don't have permission to reroll this key.",
        });
      } else {
        toast.error("Failed to Reroll Key", {
          description: errorMessage || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  const invalidateKeys = () => {
    trpcUtils.api.keys.list.invalidate();
  };

  return { ...rerollKey, invalidateKeys };
};
