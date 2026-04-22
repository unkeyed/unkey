import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

type RotatedKey = {
  keyId: string;
  key: string;
  name?: string;
};

export const useRotateKey = (onSuccess: (data: RotatedKey) => void) => {
  return trpc.key.reroll.useMutation({
    onSuccess(data) {
      onSuccess(data);
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Not Found", {
          description:
            "The key you're trying to rotate no longer exists. Please refresh and try again.",
        });
      } else if (err.data?.code === "PRECONDITION_FAILED") {
        toast.error("Rotation Not Supported", {
          description:
            err.message || "This key's configuration does not support rotation at this time.",
        });
      } else if (err.data?.code === "TOO_MANY_REQUESTS") {
        toast.error("Rate limit reached", {
          description: "Too many requests in the allowed duration. Please try again shortly.",
        });
      } else if (err.data?.code === "FORBIDDEN") {
        toast.error("Permission Denied", {
          description: "You don't have permission to rotate this key.",
        });
      } else {
        toast.error("Failed to Rotate Key", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });
};
