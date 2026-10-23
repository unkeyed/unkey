"use client";

import {
  CANCELLATION_COMMENT_MAX_LENGTH,
  CANCELLATION_REASONS,
  CANCELLATION_REASON_LABELS,
  type CancellationReason,
} from "@/lib/billing/cancellation-reasons";
import type { BillingProduct } from "@/lib/billing/products";
import { trpc } from "@/lib/trpc/client";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  FormField,
  FormTextarea,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useState } from "react";

export function CancellationFeedbackDialog({
  product,
  onClose,
}: {
  product: BillingProduct | null;
  onClose: () => void;
}) {
  const [reason, setReason] = useState<CancellationReason | null>(null);
  const [comment, setComment] = useState("");

  const send = trpc.billing.sendCancellationFeedback.useMutation({
    onSuccess: () => {
      onClose();
      toast.success("Thanks for the feedback");
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <AlertDialog
      open={product !== null}
      onOpenChange={(next) => {
        if (!next && !send.isLoading) {
          onClose();
        }
      }}
      onOpenChangeComplete={(isOpen) => {
        if (!isOpen) {
          setReason(null);
          setComment("");
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader className="place-items-start text-left">
          <AlertDialogTitle>Sorry to see you go</AlertDialogTitle>
          <AlertDialogDescription className="text-gray-12">
            If you have a minute, please tell us why you cancelled so we can do better next time. We
            read every reply.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <form
          id="cancellation-feedback"
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            if (product && reason && !send.isLoading) {
              send.mutate({ product, reason, comment });
            }
          }}
        >
          <FormField label="Reason">
            {(control) => (
              <Select items={CANCELLATION_REASON_LABELS} value={reason} onValueChange={setReason}>
                <SelectTrigger id={control.id} aria-describedby={control.describedBy} aria-required>
                  <SelectValue placeholder="Pick a reason" />
                </SelectTrigger>
                <SelectContent>
                  {CANCELLATION_REASONS.map((value) => (
                    <SelectItem key={value} value={value}>
                      {CANCELLATION_REASON_LABELS[value]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </FormField>
          <FormTextarea
            label="Anything else?"
            rows={4}
            maxLength={CANCELLATION_COMMENT_MAX_LENGTH}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
          />
        </form>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={send.isLoading}>No thanks</AlertDialogCancel>
          <Button
            type="submit"
            form="cancellation-feedback"
            variant="primary"
            disabled={reason === null || send.isLoading}
            loading={send.isLoading}
          >
            Send feedback
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
