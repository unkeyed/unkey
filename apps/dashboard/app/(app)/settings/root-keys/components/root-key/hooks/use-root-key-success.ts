import { useRef, useState } from "react";

type UseRootKeySuccessProps = {
  onClose: () => void;
};

export function useRootKeySuccess({ onClose }: UseRootKeySuccessProps) {
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);

  const dividerRef = useRef<HTMLDivElement>(null);

  const handleCloseAttempt = () => {
    setIsConfirmOpen(true);
  };

  const handleConfirmClose = () => {
    setIsConfirmOpen(false);
    onClose();
  };

  return {
    isConfirmOpen,
    setIsConfirmOpen,
    dividerRef,
    handleCloseAttempt,
    handleConfirmClose,
  };
}
