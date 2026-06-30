import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { toast } from "@unkey/ui";

type RotateKeyInput = Parameters<Unkey["keys"]["rerollKey"]>[0];
type RotateKeyResult = Awaited<ReturnType<Unkey["keys"]["rerollKey"]>>["data"];

export const useRotateKey = () => {
  return useMutation<RotateKeyResult, unknown, RotateKeyInput>({
    mutationFn: async (input) => {
      const response = await getUnkeyClient().keys.rerollKey(input);
      return response.data;
    },
    onError(err) {
      toast.error("Failed to Rotate Key", {
        description: getErrorMessage(err),
        action: {
          label: "Contact Support",
          onClick: () => window.open("mailto:support@unkey.com", "_blank"),
        },
      });
    },
  });
};
