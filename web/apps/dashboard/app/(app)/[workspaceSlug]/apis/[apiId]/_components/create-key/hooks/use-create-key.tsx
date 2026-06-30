import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { toast } from "@unkey/ui";

type CreateKeyInput = Parameters<Unkey["keys"]["createKey"]>[0];
type CreateKeyResult = Awaited<ReturnType<Unkey["keys"]["createKey"]>>["data"];

export const useCreateKey = (
  onSuccess: (data: {
    keyId: string;
    key: string;
  }) => void,
) => {
  const trpcUtils = trpc.useUtils();
  const key = useMutation<CreateKeyResult, unknown, CreateKeyInput>({
    mutationFn: async (input) => {
      const response = await getUnkeyClient().keys.createKey(input);
      return response.data;
    },
    onSuccess(data) {
      trpcUtils.api.keys.list.invalidate();
      onSuccess(data);
    },
    onError(err) {
      toast.error("Failed to Create Key", {
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
