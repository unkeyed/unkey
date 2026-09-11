import { useState } from "react";
import type { Key } from "~/components/keys-table/schema/keys.schema";
import { useRerollKey } from "~/hooks/use-reroll-key";

const DEFAULT_GRACE = "60000";

export function useRotateKey() {
  const [rotating, setRotating] = useState<Key | null>(null);
  const [grace, setGrace] = useState<string>(DEFAULT_GRACE);
  const [hasCopied, setHasCopied] = useState(false);
  const reroll = useRerollKey();

  return {
    rotating,
    grace,
    hasCopied,
    isPending: reroll.isPending,
    error: reroll.error,
    data: reroll.data,
    // Deferred a frame so the dropdown's body pointer-events lock is released
    // before the dialog mounts, which otherwise leaves the page uninteractable.
    start: (key: Key) => requestAnimationFrame(() => setRotating(key)),
    close: () => {
      if (reroll.isPending) {
        return;
      }
      setGrace(DEFAULT_GRACE);
      setHasCopied(false);
      reroll.reset();
      setRotating(null);
    },
    setGrace,
    rotate: () => {
      if (!rotating || reroll.isPending) {
        return;
      }
      reroll.mutate({ keyId: rotating.id, expiration: Number(grace) });
    },
    markCopied: () => setHasCopied(true),
  };
}

export type RotateKeyController = ReturnType<typeof useRotateKey>;
