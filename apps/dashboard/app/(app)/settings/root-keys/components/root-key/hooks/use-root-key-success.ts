import { toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { ROOT_KEY_MESSAGES } from "../constants";

type UseRootKeySuccessProps = {
  onClose: () => void;
};

export function useRootKeySuccess({ onClose }: UseRootKeySuccessProps) {
  const router = useRouter();
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<
    "close" | "create-another" | "go-to-details" | null
  >(null);
  const dividerRef = useRef<HTMLDivElement>(null);

  const handleCloseAttempt = (action: "close" | "create-another" | "go-to-details" = "close") => {
    setPendingAction(action);
    setIsConfirmOpen(true);
  };

  const handleConfirmClose = () => {
    if (!pendingAction) {
      console.error(ROOT_KEY_MESSAGES.ERROR.NO_PENDING_ACTION);
      return;
    }

    setIsConfirmOpen(false);

    try {
      // Always close the dialog first
      onClose();

      // Then execute the specific action
      switch (pendingAction) {
        case "create-another":
          // Reset form for creating another key
          break;

        case "go-to-details":
          router.push("/settings/root-keys");
          break;

        default:
          // Dialog already closed, nothing more to do
          router.push("/settings/root-keys");
          break;
      }
    } catch (_error) {
      toast.error(ROOT_KEY_MESSAGES.ERROR.ACTION_FAILED, {
        description: ROOT_KEY_MESSAGES.ERROR.ACTION_FAILED_DESCRIPTION,
      });
    } finally {
      setPendingAction(null);
    }
  };

  const handleDialogOpenChange = (open: boolean) => {
    if (!open) {
      handleCloseAttempt("close");
    }
  };

  return {
    isConfirmOpen,
    setIsConfirmOpen,
    dividerRef,
    handleCloseAttempt,
    handleConfirmClose,
    handleDialogOpenChange,
  };
}
