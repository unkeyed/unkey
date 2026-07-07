import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { toast } from "@unkey/ui";

type UpdateKeyRequest = Parameters<Unkey["keys"]["updateKey"]>[0];

type EditMetadataVariables = {
  keyId: UpdateKeyRequest["keyId"];
  meta?: UpdateKeyRequest["meta"];
};

export const useEditMetadata = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const mutation = useMutation<void, unknown, EditMetadataVariables>({
    mutationFn: async ({ keyId, meta }) => {
      await getUnkeyClient().keys.updateKey({
        keyId,
        meta,
      });
    },
    onSuccess(_, variables) {
      let description = "";
      if (variables.meta) {
        description = `Metadata for key ${variables.keyId} has been updated`;
      } else {
        description = `Metadata has been removed from key ${variables.keyId}`;
      }

      toast.success("Key Metadata Updated", {
        description,
        duration: 5000,
      });
      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      toast.error("Failed to Update Key Metadata", {
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
