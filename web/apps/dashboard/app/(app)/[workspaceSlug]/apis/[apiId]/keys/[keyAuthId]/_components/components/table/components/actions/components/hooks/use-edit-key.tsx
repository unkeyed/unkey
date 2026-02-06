import { UNNAMED_KEY } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.constants";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useEditKeyName = (onSuccess: () => void) => {
  const trpcUtils = trpc.useUtils();

  const key = trpc.key.update.name.useMutation({
    onSuccess(data) {
      const nameChange =
        data.previousName !== data.newName
          ? `from "${data.previousName || UNNAMED_KEY}" to "${data.newName || UNNAMED_KEY}"`
          : "";

      toast.success("Key Name Updated", {
        description: `Your key ${data.keyId} has been updated successfully ${nameChange}`,
        duration: 5000,
      });

      trpcUtils.api.keys.list.invalidate();
      onSuccess();
    },
    onError(err) {
      const errorMessage = err.message || "";

      if (err.data?.code === "UNPROCESSABLE_CONTENT") {
        toast.error("No Changes Detected", {
          description: "The new name must be different from the current name.",
        });
      } else if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Update Failed", {
          description: "Unable to find the key. Please refresh and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while updating your key. Please try again later or contact support at support.unkey.dev",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Configuration", {
          description: `Please check your key name. ${errorMessage}`,
        });
      } else {
        toast.error("Failed to Update Key", {
          description: errorMessage || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  return key;
};
