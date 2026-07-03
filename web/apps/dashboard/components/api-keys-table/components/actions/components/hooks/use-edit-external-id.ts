import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { toast } from "@unkey/ui";

type UpdateKeyRequest = Parameters<Unkey["keys"]["updateKey"]>[0];

type EditExternalIdVariables = {
  keyIds: UpdateKeyRequest["keyId"] | UpdateKeyRequest["keyId"][];
  externalId?: UpdateKeyRequest["externalId"];
};

const updateExternalId = async ({ keyIds, externalId }: EditExternalIdVariables) => {
  const ids = Array.isArray(keyIds) ? keyIds : [keyIds];
  if (ids.length === 0) {
    return { updatedCount: 0 };
  }

  const client = getUnkeyClient();
  await Promise.all(
    ids.map((keyId) =>
      client.keys.updateKey({
        keyId,
        externalId: externalId ?? null,
      }),
    ),
  );

  return { updatedCount: ids.length };
};

const handleExternalIdUpdateError = (err: unknown) => {
  toast.error("Failed to Update Key External ID", {
    description: getErrorMessage(err),
    action: {
      label: "Contact Support",
      onClick: () => window.open("mailto:support@unkey.com", "_blank"),
    },
  });
};

export const useEditExternalId = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const mutation = useMutation<
    Awaited<ReturnType<typeof updateExternalId>>,
    unknown,
    EditExternalIdVariables
  >({
    mutationFn: updateExternalId,
    onSuccess(_, variables) {
      let description = "";
      const keyId = Array.isArray(variables.keyIds) ? variables.keyIds[0] : variables.keyIds;

      if (variables.externalId) {
        description = `Identity for key ${keyId} has been updated`;
      } else {
        description = `Identity has been removed from key ${keyId}`;
      }
      toast.success("Key External ID Updated", {
        description,
        duration: 5000,
      });

      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      handleExternalIdUpdateError(err);
    },
  });

  return mutation;
};

export const useBatchEditExternalId = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const mutation = useMutation<
    Awaited<ReturnType<typeof updateExternalId>>,
    unknown,
    EditExternalIdVariables
  >({
    mutationFn: updateExternalId,
    onSuccess(data, variables) {
      const updatedCount = data.updatedCount;
      let description = "";

      if (variables.externalId) {
        description = `Identity has been updated for ${updatedCount} ${
          updatedCount === 1 ? "key" : "keys"
        }`;
      } else {
        description = `Identity has been removed from ${updatedCount} ${
          updatedCount === 1 ? "key" : "keys"
        }`;
      }
      toast.success("Key External ID Updated", {
        description,
        duration: 5000,
      });

      // Show warning if some keys were not found (if that info is available in the response)
      const missingCount = Array.isArray(variables.keyIds)
        ? variables.keyIds.length - updatedCount
        : 0;

      if (missingCount > 0) {
        toast.warning("Some Keys Not Found", {
          description: `${missingCount} ${
            missingCount === 1 ? "key was" : "keys were"
          } not found and could not be updated.`,
          duration: 7000,
        });
      }

      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      handleExternalIdUpdateError(err);
    },
  });

  return mutation;
};
