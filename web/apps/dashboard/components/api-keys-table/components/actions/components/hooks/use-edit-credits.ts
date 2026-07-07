import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { toast } from "@unkey/ui";

type UpdateKeyRequest = Parameters<Unkey["keys"]["updateKey"]>[0];

type EditCreditsVariables = {
  keyId: UpdateKeyRequest["keyId"];
  credits?: UpdateKeyRequest["credits"];
};

export const useEditCredits = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const mutation = useMutation<void, unknown, EditCreditsVariables>({
    mutationFn: async ({ keyId, credits }) => {
      await getUnkeyClient().keys.updateKey({
        keyId,
        credits,
      });
    },
    onSuccess(_, variables) {
      const remainingChange = variables.credits
        ? `with ${variables.credits.remaining} uses remaining`
        : "with limits disabled";

      toast.success("Key Limits Updated", {
        description: `Your key ${variables.keyId} has been updated successfully ${remainingChange}`,
        duration: 5000,
      });
      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      toast.error("Failed to Update Key Limits", {
        description: getErrorMessage(err),
        action: {
          label: "Contact Support",
          onClick: () => window.open("mailto:support@unkey.com", "_blank"),
        },
      });
    },
  });
  return mutation;
};
