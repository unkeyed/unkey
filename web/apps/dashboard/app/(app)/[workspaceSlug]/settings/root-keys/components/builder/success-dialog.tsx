"use client";

import { SecretKey } from "@/components/secret-key";
import { Button, Dialog, DialogContent, DialogTitle } from "@unkey/ui";

type SuccessDialogProps = {
  secret: string;
  onDone: () => void;
};

export function SuccessDialog({ secret, onDone }: SuccessDialogProps) {
  return (
    <Dialog
      open
      onOpenChange={(open, details) => {
        if (open) {
          return;
        }
        if (details.reason === "escape-key") {
          details.cancel();
          return;
        }
        onDone();
      }}
    >
      <DialogContent
        preventOutsideClose
        className="w-full max-w-[560px] gap-4 rounded-2xl! border-grayA-4 p-6"
      >
        <DialogTitle>Root Key created</DialogTitle>
        <p className="text-[13px] leading-5 text-gray-11">
          This is the only time the full key is shown. Store it somewhere safe — you can roll it
          later, but not read it again.
        </p>
        <SecretKey value={secret} title="Root Key" />
        <div className="flex items-center justify-end pt-1">
          <Button type="button" variant="primary" size="md" onClick={onDone}>
            Done
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
