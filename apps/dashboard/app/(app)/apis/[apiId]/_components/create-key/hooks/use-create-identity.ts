import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useCreateIdentity = (
  onSuccess?: (data: { identityId: string; externalId: string }) => void,
) => {
  const trpcUtils = trpc.useUtils();

  const createIdentityMutation = trpc.identity.create.useMutation({
    onSuccess(data) {
      toast.success("Identity Created", {
        description: `Identity "${data.externalId}" has been created successfully`,
        duration: 5000,
      });

      trpcUtils.identity.query.invalidate();
      trpcUtils.identity.search.invalidate();

      if (onSuccess) {
        onSuccess(data);
      }
    },

    onError(err) {
      if (err.data?.code === "CONFLICT") {
        toast.error("Identity Already Exists", {
          description: "An identity with this external ID already exists in your workspace.",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Input", {
          description: err.message || "Please check your input and try again.",
        });
      } else {
        toast.error("Failed to Create Identity", {
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

  return createIdentityMutation;
};
