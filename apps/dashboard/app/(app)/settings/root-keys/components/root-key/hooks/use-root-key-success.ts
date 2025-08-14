import { toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { ROOT_KEY_CONSTANTS, ROOT_KEY_MESSAGES } from "../constants";

type UseRootKeySuccessProps = {
  keyValue?: string;
  onClose: () => void;
};

export function useRootKeySuccess({ keyValue, onClose }: UseRootKeySuccessProps) {
  const router = useRouter();
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);
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

  const snippet = `curl -XPOST '${ROOT_KEY_CONSTANTS.API_URL}/v1/keys.createKey' \\
    -H 'Authorization: Bearer ${keyValue}' \\
    -H 'Content-Type: application/json' \\
    -d '{
      "prefix": "hello",
      "apiId": "<API_ID>"
    }'`;

  const split = keyValue?.split("_") ?? [];
  const maskedKey =
    split.length >= 2
      ? `${split.at(0)}_${"*".repeat(split.at(1)?.length ?? 0)}`
      : "*".repeat(split.at(0)?.length ?? 0);

  return {
    showKeyInSnippet,
    setShowKeyInSnippet,
    isConfirmOpen,
    setIsConfirmOpen,
    dividerRef,
    handleCloseAttempt,
    handleConfirmClose,
    handleDialogOpenChange,
    snippet,
    maskedKey,
  };
}
