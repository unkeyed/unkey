import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";

export const useCreateKey = (
  onSuccess: (data: {
    keyId: `key_${string}`;
    key: string;
    name?: string;
  }) => void,
) => {
  const trpcUtils = trpc.useUtils();
  const key = trpc.key.create.useMutation({
    onSuccess(data) {
      toast.success("Key Created Successfully", {
        description: `Your key ${data.keyId} has been created and is ready to use`,
        duration: 5000,
      });
      trpcUtils.api.keys.list.invalidate();
      onSuccess(data);
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Creation Failed", {
          description:
            "Unable to find the correct API configuration. Please refresh and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while creating your key. Please try again later or contact support at support.unkey.dev",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Configuration", {
          description: `Please check your key settings. ${err.message || ""}`,
        });
      } else {
        toast.error("Failed to Create Key", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("https://support.unkey.dev", "_blank"),
          },
        });
      }
    },
  });

  return key;
};
