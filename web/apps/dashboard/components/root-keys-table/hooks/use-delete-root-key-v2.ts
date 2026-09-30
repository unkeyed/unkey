import { ROOT_KEYS_V2_QUERY_KEY, deleteRootKey } from "@/lib/root-keys-api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "@unkey/ui";

export const useDeleteRootKeyV2 = (
  onSuccess: (data: { keyIds: string[]; message: string }) => void,
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ keyIds }: { keyIds: string[] }) => {
      await Promise.all(keyIds.map((keyId) => deleteRootKey({ keyId })));
    },
    onSuccess: async (_, variables) => {
      await queryClient.invalidateQueries({ queryKey: ROOT_KEYS_V2_QUERY_KEY });
      toast.success("Root key deleted", {
        description:
          "The root key has been permanently deleted and can no longer create resources.",
      });
      onSuccess({
        keyIds: variables.keyIds,
        message: "Root key deleted successfully",
      });
    },
    onError: (error) => {
      toast.error("Failed to revoke root key", {
        description: error instanceof Error ? error.message : "Please try again later.",
      });
    },
  });
};
