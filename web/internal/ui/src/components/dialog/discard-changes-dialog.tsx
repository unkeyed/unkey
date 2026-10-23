"use client";

import { Button } from "../buttons/button";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "./alert-dialog";

export type DiscardChangesCopy = {
  title: string;
  description: string;
  keepLabel: string;
  discardLabel: string;
};

const DEFAULT_COPY: DiscardChangesCopy = {
  title: "Discard unsaved changes?",
  description: "You'll lose the changes you made.",
  keepLabel: "Keep editing",
  discardLabel: "Discard changes",
};

export type DiscardChangesDialogProps = {
  open: boolean;
  onKeepEditing: () => void;
  onDiscard: () => void;
  copy?: DiscardChangesCopy;
  /** Shows the discard as running. The dialog cannot be dismissed until it ends. */
  loading?: boolean;
};

export function DiscardChangesDialog({
  open,
  onKeepEditing,
  onDiscard,
  copy = DEFAULT_COPY,
  loading = false,
}: DiscardChangesDialogProps) {
  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !loading) {
          onKeepEditing();
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{copy.title}</AlertDialogTitle>
          <AlertDialogDescription>{copy.description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={loading}>{copy.keepLabel}</AlertDialogCancel>
          <Button variant="primary" size="md" color="danger" loading={loading} onClick={onDiscard}>
            {copy.discardLabel}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
