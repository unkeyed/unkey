import { AlertTriangle } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "~/components/ui/alert-dialog";
import { Checkbox } from "~/components/ui/checkbox";

type DeleteKeyDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
};

export function DeleteKeyDialog({ open, onOpenChange, onConfirm }: DeleteKeyDialogProps) {
  const [confirmed, setConfirmed] = useState(false);
  const checkboxId = useId();

  useEffect(() => {
    if (!open) {
      setConfirmed(false);
    }
  }, [open]);

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete this key?</AlertDialogTitle>
          <AlertDialogDescription>
            Anything using this key will stop working immediately.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <Alert variant="destructive" className="mt-4">
          <AlertTriangle />
          <AlertTitle>This action is permanent.</AlertTitle>
          <AlertDescription>There's no way to recover a deleted key.</AlertDescription>
        </Alert>

        <label
          htmlFor={checkboxId}
          className="mt-4 flex cursor-pointer items-center gap-2 text-gray-12 text-sm"
        >
          <Checkbox
            id={checkboxId}
            checked={confirmed}
            onCheckedChange={(value) => setConfirmed(value === true)}
          />
          I understand this can't be undone.
        </label>

        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={!confirmed} onClick={onConfirm}>
            Delete key
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
