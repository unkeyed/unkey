"use client";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

type Props = {
  open: boolean;
  setOpen: (open: boolean) => void;
  onClick: () => void;
  lastUsed: number;
};

export function RerollConfirmationDialog({ open, setOpen, onClick, lastUsed }: Props) {
  return (
    <Dialog open={open} onOpenChange={(o: boolean) => setOpen(o)}>
      <DialogContent className="border-alert">
        <DialogHeader>
          <DialogTitle>Reroll Key</DialogTitle>
          <DialogDescription>
            Make sure to replace it in your system before it expires. This action cannot be undone.
          </DialogDescription>
        </DialogHeader>

        <p className="text-sm text-content mt-1 mb-2">
          {lastUsed
            ? `This key was used for the last time on ${new Date(lastUsed).toString()}`
            : "This key was never used."}
        </p>

        <Alert variant="alert">
          <AlertTitle>Warning</AlertTitle>
          <AlertDescription>This action is not reversible. Please be certain.</AlertDescription>
        </Alert>

        <DialogFooter className="justify-end">
          <Button
            aria-label="Cancel"
            type="button"
            onClick={() => setOpen(!open)}
            variant="secondary"
          >
            Cancel
          </Button>
          <Button aria-label="Reroll Key Submit" type="submit" variant="alert" onClick={onClick}>
            Reroll Key
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
