"use client";

import {
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import type { ReactNode } from "react";
import { FormProvider, type UseFormReturn, useFormState } from "react-hook-form";
import type { ConnectionFormValues } from "../../model/form";

export function ConnectionForm({
  title,
  isOpen,
  onClose,
  form,
  footer,
  children,
}: {
  title: string;
  isOpen: boolean;
  onClose: () => void;
  form: UseFormReturn<ConnectionFormValues>;
  footer?: ReactNode;
  children: ReactNode;
}) {
  const { isDirty } = useFormState({ control: form.control });

  return (
    <SlidePanel dirty={isDirty} isOpen={isOpen} onClose={onClose}>
      <SlidePanelHeader>
        <SlidePanelTitle>{title}</SlidePanelTitle>
        <SlidePanelCloseButton className="mt-0.5" />
      </SlidePanelHeader>
      <SlidePanelContent>
        <FormProvider {...form}>
          <form
            onSubmit={(event) => event.preventDefault()}
            className="flex h-full flex-col gap-3 overflow-y-auto px-6 pt-1 pb-8"
          >
            {children}
          </form>
        </FormProvider>
      </SlidePanelContent>
      {footer ? (
        <SlidePanelFooter className="flex items-center justify-between gap-3">
          {footer}
        </SlidePanelFooter>
      ) : null}
    </SlidePanel>
  );
}
