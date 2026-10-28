"use client";

import {
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
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
  onSubmit,
  children,
}: {
  title: string;
  isOpen: boolean;
  onClose: () => void;
  form: UseFormReturn<ConnectionFormValues>;
  onSubmit: (values: ConnectionFormValues) => Promise<void>;
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
            onSubmit={form.handleSubmit(onSubmit)}
            className="flex h-full flex-col gap-3 overflow-y-auto px-6 pt-1 pb-8"
          >
            {children}
          </form>
        </FormProvider>
      </SlidePanelContent>
    </SlidePanel>
  );
}
