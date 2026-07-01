import { UNNAMED_KEY } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.constants";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { toast } from "@unkey/ui";

type UpdateKeyRequest = Parameters<Unkey["keys"]["updateKey"]>[0];

type EditKeyNameVariables = {
  keyId: UpdateKeyRequest["keyId"];
  name?: string | undefined;
  originalName?: string | undefined;
};

type EditKeyNameResult = {
  keyId: string;
  previousName?: string | null;
  newName?: string | null;
};

export const useEditKeyName = (onSuccess: () => void) => {
  const trpcUtils = trpc.useUtils();

  const key = useMutation<EditKeyNameResult, unknown, EditKeyNameVariables>({
    mutationFn: async ({ keyId, name, originalName }) => {
      const normalizedName = name?.trim() || null;
      await getUnkeyClient().keys.updateKey({
        keyId,
        name: normalizedName,
      });

      return {
        keyId,
        previousName: originalName?.trim() || null,
        newName: normalizedName,
      };
    },
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
      toast.error("Failed to Update Key", {
        description: getErrorMessage(err),
        action: {
          label: "Contact Support",
          onClick: () => window.open("mailto:support@unkey.com", "_blank"),
        },
      });
    },
  });

  return key;
};
