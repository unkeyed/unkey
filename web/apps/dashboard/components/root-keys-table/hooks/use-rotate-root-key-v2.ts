import { rerollRootKey } from "@/lib/root-keys-api";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";

export const useRotateRootKeyV2 = () =>
  useMutation({
    mutationFn: rerollRootKey,
    onError: (error) => {
      toast.error("Failed to rotate root key", {
        description: error instanceof Error ? error.message : "Please try again later.",
      });
    },
  });
