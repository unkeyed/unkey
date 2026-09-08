import { useState } from "react";

type CloseGateOptions = {
  hasSecret: boolean;
  hasCopied: boolean;
  onClose: () => void;
};

type CloseGate = {
  tryClose: () => void;
  discardConfirm: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onDiscard: () => void;
  };
};

/**
 * Gates dialog dismissal when a freshly-revealed secret hasn't been copied yet.
 * Routes ESC, outside-click, and the X button through `tryClose`; pops a
 * confirmation AlertDialog if the user is about to lose the only chance to
 * grab the secret.
 */
export function useSecretCloseGate({ hasSecret, hasCopied, onClose }: CloseGateOptions): CloseGate {
  const [alertOpen, setAlertOpen] = useState(false);
  const requiresConfirm = hasSecret && !hasCopied;

  const tryClose = () => {
    if (requiresConfirm) {
      setAlertOpen(true);
    } else {
      onClose();
    }
  };

  const onDiscard = () => {
    setAlertOpen(false);
    onClose();
  };

  return {
    tryClose,
    discardConfirm: { open: alertOpen, onOpenChange: setAlertOpen, onDiscard },
  };
}
