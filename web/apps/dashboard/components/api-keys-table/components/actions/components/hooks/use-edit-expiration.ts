import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { toast } from "@unkey/ui";

type UpdateKeyRequest = Parameters<Unkey["keys"]["updateKey"]>[0];

type EditExpirationVariables = {
  keyId: UpdateKeyRequest["keyId"];
  expires?: Date | null;
};

export const useEditExpiration = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const mutation = useMutation<void, unknown, EditExpirationVariables>({
    mutationFn: async ({ keyId, expires }) => {
      await getUnkeyClient().keys.updateKey({
        keyId,
        expires: expires instanceof Date ? expires.getTime() : expires,
      });
    },
    onSuccess(_, variables) {
      let description = "";
      if (variables.expires) {
        description = `Your key ${
          variables.keyId
        } has been updated to expire on ${variables.expires.toLocaleString()}`;
      } else {
        description = `Expiration has been disabled for key ${variables.keyId}`;
      }

      toast.success("Key Expiration Updated", {
        description,
        duration: 5000,
      });
      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      toast.error("Failed to Update Key Expiration", {
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
