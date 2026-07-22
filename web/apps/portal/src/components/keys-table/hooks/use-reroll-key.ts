import { useMutation, useQueryClient } from "@tanstack/react-query";
import { rerollKey } from "~/lib/portal-api";
import type { RerollKeyRequest, RerollKeyResult } from "../schema/keys.schema";
import { keysListQueryKey } from "./queries/use-keys-list-query";

/**
 * Rerolls a key via `v2/portal.rerollKey` and invalidates the keys list so the
 * new prefix/expiry appear once the mutation settles. The caller awaits
 * `mutateAsync` to receive the one-time plaintext secret to display.
 */
export function useRerollKey() {
  const queryClient = useQueryClient();

  return useMutation<RerollKeyResult, Error, RerollKeyRequest>({
    mutationFn: (input) => rerollKey({ data: input }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keysListQueryKey });
    },
  });
}
