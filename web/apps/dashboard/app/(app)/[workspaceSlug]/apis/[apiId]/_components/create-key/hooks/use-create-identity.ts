import { useCreateIdentityMutation } from "@/lib/identities-query";
import { getErrorMessage } from "@/lib/unkey-client";
import { BadRequestErrorResponse, ConflictErrorResponse } from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";

export const useCreateIdentity = (
  onSuccess?: (data: { identityId: string; externalId: string }) => void,
) => {
  const createIdentityMutation = useCreateIdentityMutation();

  return {
    ...createIdentityMutation,
    mutate: (input: Parameters<typeof createIdentityMutation.mutate>[0]) =>
      createIdentityMutation.mutate(input, {
        onSuccess(data) {
          toast.success("Identity Created", {
            description: `Identity "${data.externalId}" has been created successfully`,
            duration: 5000,
          });

          if (onSuccess) {
            onSuccess(data);
          }
        },
        onError(error) {
          if (error instanceof ConflictErrorResponse) {
            toast.error("Identity Already Exists", {
              description: "An identity with this external ID already exists in your workspace.",
            });
          } else if (error instanceof BadRequestErrorResponse) {
            toast.error("Invalid Input", {
              description: getErrorMessage(error, "Please check your input and try again."),
            });
          } else {
            toast.error("Failed to Create Identity", {
              description: getErrorMessage(error),
              action: {
                label: "Contact Support",
                onClick: () => window.open("mailto:support@unkey.com", "_blank"),
              },
            });
          }
        },
      }),
  };
};
