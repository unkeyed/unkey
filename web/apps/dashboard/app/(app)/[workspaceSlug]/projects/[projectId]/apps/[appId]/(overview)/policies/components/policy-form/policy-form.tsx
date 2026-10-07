"use client";

import { GuardedSlidePanel } from "@/components/guarded-slide-panel";
import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { IconArrowUpRightOutline12 } from "@unkey/icons";
import {
  Button,
  FormInput,
  FormSelect,
  SlidePanelClose,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import Link from "next/link";
import { type ReactNode, useState } from "react";
import { Controller, FormProvider, type UseFormReturn, useFormState } from "react-hook-form";
import {
  POLICY_KINDS,
  POLICY_TYPES,
  type PolicyFormValues,
  type PolicyType,
  hasConfigErrors,
} from "../../policy-kinds";
import { POLICY_DOCS_URL } from "../../policy-kinds/types";
import { RuleCard } from "./rule";

const TYPE_OPTIONS = POLICY_TYPES.map((type) => ({ value: type, label: POLICY_KINDS[type].label }));

function focusFirstError() {
  const container = document.querySelector('[aria-invalid="true"], [data-error="true"]');
  if (!(container instanceof HTMLElement)) {
    return;
  }
  const inner = container.matches("input, button, select, textarea")
    ? null
    : container.querySelector("button, input, select, textarea");
  const target = inner instanceof HTMLElement ? inner : container;
  target.focus();
  target.scrollIntoView({ behavior: "smooth", block: "center" });
}

export function PolicyForm({
  title,
  submitLabel,
  isOpen,
  onClose,
  form,
  onSubmit,
  onTypeChange,
  children,
}: {
  title: string;
  submitLabel: string;
  isOpen: boolean;
  onClose: () => void;
  form: UseFormReturn<PolicyFormValues>;
  onSubmit: (values: PolicyFormValues) => void | Promise<void>;
  onTypeChange?: (type: PolicyType) => void;
  children: ReactNode;
}) {
  const [configOpen, setConfigOpen] = useState(false);
  const { isDirty } = useFormState({ control: form.control });

  return (
    <GuardedSlidePanel dirty={isDirty} isOpen={isOpen} onClose={onClose}>
      <SlidePanelHeader>
        <div className="flex flex-col gap-0.5">
          <SlidePanelTitle>{title}</SlidePanelTitle>
          <SlidePanelDescription>{NEXT_DEPLOY}.</SlidePanelDescription>
        </div>
        <SlidePanelCloseButton className="mt-0.5" />
      </SlidePanelHeader>
      <SlidePanelContent>
        <FormProvider {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit, (errors) => {
              if (hasConfigErrors(errors)) {
                setConfigOpen(true);
              }
              setTimeout(focusFirstError, 0);
            })}
            className="flex h-full flex-col"
          >
            <div className="flex flex-1 flex-col gap-8 overflow-y-auto px-6 pt-4 pb-8">
              <div className="grid grid-cols-2 gap-4">
                <Controller
                  control={form.control}
                  name="name"
                  render={({ field, fieldState }) => (
                    <FormInput
                      label="Name"
                      placeholder="e.g. API Key Auth, Rate Limit Public"
                      value={field.value}
                      onChange={field.onChange}
                      error={fieldState.error?.message}
                    />
                  )}
                />
                <Controller
                  control={form.control}
                  name="type"
                  render={({ field }) => (
                    <FormSelect
                      label="Type"
                      options={TYPE_OPTIONS}
                      value={field.value}
                      onValueChange={(next) => {
                        const type = POLICY_TYPES.find((t) => t === next);
                        if (onTypeChange && type) {
                          onTypeChange(type);
                        }
                      }}
                      disabled={!onTypeChange}
                    />
                  )}
                />
              </div>
              <RuleCard configOpen={configOpen} onConfigOpenChange={setConfigOpen} />
              {children}
            </div>
            <div className="flex items-center justify-between gap-3 border-t bg-grayA-2 px-4 py-3">
              <Button
                variant="ghost"
                size="sm"
                className="text-gray-11 hover:text-gray-12 [&_svg]:size-3"
                render={<Link href={POLICY_DOCS_URL} target="_blank" rel="noopener noreferrer" />}
              >
                Documentation
                <IconArrowUpRightOutline12 />
              </Button>
              <div className="flex items-center gap-2">
                <SlidePanelClose render={<Button type="button" variant="outline" size="sm" />}>
                  Cancel
                </SlidePanelClose>
                <Button
                  type="submit"
                  variant="primary"
                  size="sm"
                  className="px-3"
                  loading={form.formState.isSubmitting}
                >
                  {submitLabel}
                </Button>
              </div>
            </div>
          </form>
        </FormProvider>
      </SlidePanelContent>
    </GuardedSlidePanel>
  );
}
