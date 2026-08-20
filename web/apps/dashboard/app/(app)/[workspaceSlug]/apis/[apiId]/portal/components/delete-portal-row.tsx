"use client";

import { useDeletePortal } from "@/lib/portal/use-portal";
import type { Portal } from "@unkey/api/models/components";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer, Input, SettingsZoneRow, toast } from "@unkey/ui";
import { useState } from "react";
import { useForm } from "react-hook-form";

type ConfirmationForm = { confirmation: string };

/**
 * Deleting is not disabling: the row is gone, every live session is revoked by
 * `RevokePortalSessionsByPortal`, and the slug returns to the pool. The
 * type-to-confirm string is the slug because a portal carries no name.
 */
export function DeletePortalRow({ portal, keyAuthId }: { portal: Portal; keyAuthId: string }) {
  const [isOpen, setIsOpen] = useState(false);
  const deletePortal = useDeletePortal(keyAuthId);

  // No resolver: the only gate is the exact-match comparison below, so
  // validating the field as well would run a schema nothing reads.
  const {
    register,
    watch,
    reset,
    handleSubmit,
    formState: { isSubmitting },
  } = useForm<ConfirmationForm>({
    defaultValues: { confirmation: "" },
  });

  const confirmed = watch("confirmation") === portal.slug;

  // The form outlives the dialog, so without this a reopen would arrive with
  // the destructive button already armed from the last time it was typed.
  const setOpen = (open: boolean) => {
    if (!open) {
      reset({ confirmation: "" });
    }
    setIsOpen(open);
  };

  const onSubmit = async (values: ConfirmationForm) => {
    // Re-checked here rather than trusting the button's `disabled` attribute:
    // a form can be submitted by other means than clicking it.
    if (values.confirmation !== portal.slug) {
      return;
    }
    try {
      await deletePortal.mutateAsync({ portal: portal.id });
      setOpen(false);
      toast.success("Customer portal deleted");
    } catch {
      // The hook already surfaced the failure; the dialog stays open so the
      // operator can retry without retyping the slug.
    }
  };

  return (
    <>
      <SettingsZoneRow
        title="Delete this portal"
        description="Permanently remove the portal, end every live end-user session, and free the slug for reuse."
        action={{
          label: "Delete portal",
          onClick: () => setOpen(true),
        }}
      />

      <DialogContainer
        isOpen={isOpen}
        onOpenChange={setOpen}
        title="Delete customer portal"
        subTitle="Permanently remove this portal and end every live session"
        footer={
          <div className="flex w-full flex-col items-center justify-center gap-2">
            <Button
              type="submit"
              form="delete-portal-form"
              variant="primary"
              color="danger"
              size="xlg"
              className="w-full rounded-lg"
              disabled={!confirmed || isSubmitting}
              loading={isSubmitting}
              loadingLabel="Deleting customer portal"
            >
              Delete portal
            </Button>
            <div className="text-xs text-gray-9">
              This action cannot be undone – proceed with caution
            </div>
          </div>
        }
      >
        <div className="flex items-center gap-4 rounded-xl border border-errorA-3 bg-errorA-2 px-[22px] py-6 dark:bg-black">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-error-9">
            <TriangleWarning2 iconSize="sm-regular" className="text-white" />
          </div>
          <div className="text-[13px] leading-6 text-error-12">
            <span className="font-medium">Warning:</span> deleting{" "}
            <span className="font-medium">{portal.slug}</span> is permanent. Every live end-user
            session ends immediately, and the slug becomes available for reuse. Your users' API keys
            keep working.
          </div>
        </div>
        <form id="delete-portal-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="mt-4 flex flex-col gap-1">
            <p className="text-[13px] text-gray-11">
              Type <span className="font-medium text-gray-12">{portal.slug}</span> to confirm
            </p>
            <Input
              aria-label="Portal slug confirmation"
              placeholder={`Enter "${portal.slug}" to confirm`}
              {...register("confirmation")}
            />
          </div>
        </form>
      </DialogContainer>
    </>
  );
}
