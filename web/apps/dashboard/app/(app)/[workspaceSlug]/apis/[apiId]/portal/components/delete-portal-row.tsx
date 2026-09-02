"use client";

import { useDeletePortal } from "@/lib/portal/use-portal";
import type { Portal } from "@unkey/api/models/components";
import { Button, DialogContainer, Input, SettingsZoneRow, toast } from "@unkey/ui";
import { IconTriangleWarningOutline12 } from "nucleo-ui-outline-12";
import { useState } from "react";
import { useForm } from "react-hook-form";

type ConfirmationForm = { confirmation: string };

// Unlike disabling, deleting revokes every live session and returns the slug to
// the pool.
export function DeletePortalRow({ portal, keyAuthId }: { portal: Portal; keyAuthId: string }) {
  const [isOpen, setIsOpen] = useState(false);
  const deletePortal = useDeletePortal(keyAuthId);

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

  // The form outlives the dialog, so a reopen would otherwise arrive with the
  // destructive button already armed.
  const setOpen = (open: boolean) => {
    if (!open) {
      reset({ confirmation: "" });
    }
    setIsOpen(open);
  };

  const onSubmit = async (values: ConfirmationForm) => {
    // A form can be submitted without clicking the disabled button.
    if (values.confirmation !== portal.slug) {
      return;
    }
    try {
      await deletePortal.mutateAsync({ portal: portal.id });
      setOpen(false);
      toast.success("Customer portal deleted");
    } catch {
      // The hook surfaced the failure; keep the dialog open to retry.
    }
  };

  return (
    <>
      <SettingsZoneRow
        title="Delete this portal"
        description="Permanently remove the portal, end every live user session, and free the slug for reuse."
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
            <IconTriangleWarningOutline12 className="text-white" />
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
