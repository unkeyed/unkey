import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { toast } from "@unkey/ui";

type UpdateKeyRequest = Parameters<Unkey["keys"]["updateKey"]>[0];

type UpdateKeyStatusVariables = {
  keyIds: UpdateKeyRequest["keyId"][];
  enabled: NonNullable<UpdateKeyRequest["enabled"]>;
};

type UpdateKeyStatusResult = {
  enabled: boolean;
  updatedKeyIds: string[];
};

async function updateKeysStatus({
  keyIds,
  enabled,
}: UpdateKeyStatusVariables): Promise<UpdateKeyStatusResult> {
  if (keyIds.length === 0) {
    return {
      enabled,
      updatedKeyIds: [],
    };
  }

  const updatedKeyIds = await Promise.all(
    keyIds.map(async (keyId) => {
      await getUnkeyClient().keys.updateKey({ keyId, enabled });
      return keyId;
    }),
  );

  return {
    enabled,
    updatedKeyIds,
  };
}

function showUpdateKeyStatusError(error: unknown) {
  toast.error("Failed to Update Key Status", {
    description: getErrorMessage(error),
    action: {
      label: "Contact Support",
      onClick: () => window.open("mailto:support@unkey.com", "_blank"),
    },
  });
}

export const useUpdateKeyStatus = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();

  const updateKeyEnabled = useMutation<UpdateKeyStatusResult, unknown, UpdateKeyStatusVariables>({
    mutationFn: updateKeysStatus,
    onSuccess(data) {
      if (data.updatedKeyIds.length === 0) {
        return;
      }

      toast.success(`Key ${data.enabled ? "Enabled" : "Disabled"}`, {
        description: `Your key ${data.updatedKeyIds[0]} has been ${
          data.enabled ? "enabled" : "disabled"
        } successfully`,
        duration: 5000,
      });
      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      showUpdateKeyStatusError(err);
    },
  });

  return updateKeyEnabled;
};

export const useBatchUpdateKeyStatus = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();

  const updateMultipleKeysEnabled = useMutation<
    UpdateKeyStatusResult,
    unknown,
    UpdateKeyStatusVariables
  >({
    mutationFn: updateKeysStatus,
    onSuccess(data) {
      const updatedCount = data.updatedKeyIds.length;
      if (updatedCount === 0) {
        return;
      }

      toast.success(`Keys ${data.enabled ? "Enabled" : "Disabled"}`, {
        description: `${updatedCount} ${
          updatedCount === 1 ? "key has" : "keys have"
        } been ${data.enabled ? "enabled" : "disabled"} successfully`,
        duration: 5000,
      });

      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err) {
      showUpdateKeyStatusError(err);
    },
  });

  return updateMultipleKeysEnabled;
};
