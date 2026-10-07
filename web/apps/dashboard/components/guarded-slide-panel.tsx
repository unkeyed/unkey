"use client";

import { SlidePanel, type SlidePanelProps } from "@unkey/ui";
import { useState } from "react";
import { DiscardChangesDialog } from "./discard-changes-dialog";

export function GuardedSlidePanel({
  dirty,
  isOpen,
  onClose,
  children,
  ...props
}: SlidePanelProps & { dirty: boolean }) {
  const [confirming, setConfirming] = useState(false);
  if (!isOpen && confirming) {
    setConfirming(false);
  }

  return (
    <SlidePanel
      {...props}
      isOpen={isOpen}
      onClose={() => {
        if (dirty) {
          setConfirming(true);
        } else {
          onClose();
        }
      }}
    >
      {children}
      <DiscardChangesDialog
        open={confirming}
        onKeepEditing={() => setConfirming(false)}
        onDiscard={() => {
          setConfirming(false);
          onClose();
        }}
      />
    </SlidePanel>
  );
}
