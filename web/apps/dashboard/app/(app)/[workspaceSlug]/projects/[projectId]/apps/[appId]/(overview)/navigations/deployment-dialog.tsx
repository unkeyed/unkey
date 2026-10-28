"use client";

import { Button } from "@unkey/ui";
import dynamic from "next/dynamic";
import type { FormEventHandler, ReactNode } from "react";
import { useId } from "react";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

const DIALOG_TITLE = "Create Deployment";

type DeploymentDialogProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  subTitle: string;
  submitLabel: string;
  canSubmit: boolean;
  isSubmitting: boolean;
  onSubmit: FormEventHandler<HTMLFormElement>;
  children: ReactNode;
};

export function DeploymentDialog({
  isOpen,
  onOpenChange,
  subTitle,
  submitLabel,
  canSubmit,
  isSubmitting,
  onSubmit,
  children,
}: DeploymentDialogProps) {
  const formId = useId();

  return (
    <DynamicDialogContainer
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      title={DIALOG_TITLE}
      subTitle={subTitle}
      preventAutoFocus
      footer={
        <div className="w-full flex flex-col gap-2 items-center justify-center">
          <Button
            type="submit"
            form={formId}
            variant="primary"
            size="xlg"
            disabled={!canSubmit || isSubmitting}
            loading={isSubmitting}
            className="w-full rounded-lg"
          >
            {submitLabel}
          </Button>
        </div>
      }
    >
      <form id={formId} onSubmit={onSubmit} className="flex flex-col gap-5 py-2">
        {children}
      </form>
    </DynamicDialogContainer>
  );
}
